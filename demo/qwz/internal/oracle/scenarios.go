package oracle

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/osm/quake/demo/qwd"
	"github.com/osm/quake/demo/qwz"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/protocol/qizmo"
	"github.com/osm/quake/qizmo/assets"
	"github.com/osm/quake/qizmo/freq"
)

const (
	SeedQWZSHA256 = "688eb9dbaf17bc1a9dc6d2ad5a4f9a14123f491d61a64a3c32973d00a976d2f3"
	BaseQWDSHA256 = "c9a50bee84e50b17d5d67b5f83741a89bd1f0479d4c10178712c254abd568370"
)

var (
	playerAnchor = []byte{
		protocol.SVCPlayerInfo, 0x01, 0x03, 0x00,
		0x19, 0x18, 0xe6, 0xf9, 0x40, 0x03, 0x0e,
		0x23, 0x81, 0x8b, 0x06, 0x79, 0x39, 0x0d,
	}
	packetEntitiesAnchor = []byte{
		protocol.SVCPacketEntities,
		0x40, 0x08, 0x80, 0x02,
		0x41, 0x08, 0x80, 0x02,
		0x69, 0x08, 0x90, 0x01,
		0x6a, 0x08, 0x80, 0x01,
		0x6b, 0x08, 0x80, 0x01,
		0x00, 0x00,
	}
)

type Scenario struct {
	Name   string
	Family string
	QWD    []byte
}

type record struct {
	start   int
	end     int
	command byte
}

func Build(seedQWZ []byte) ([]Scenario, error) {
	if got := SHA256(seedQWZ); got != SeedQWZSHA256 {
		return nil, fmt.Errorf("seed QWZ checksum %s, want %s", got, SeedQWZSHA256)
	}
	ft, err := freq.NewTables(freq.DefaultCompressDat)
	if err != nil {
		return nil, fmt.Errorf("frequency tables: %w", err)
	}
	seedQWD, err := qwz.Decode(seedQWZ, ft, assets.Embedded())
	if err != nil {
		return nil, fmt.Errorf("decode seed: %w", err)
	}
	records, err := parseRecords(seedQWD)
	if err != nil {
		return nil, err
	}
	if len(records) <= 50 {
		return nil, fmt.Errorf("seed has %d records, need at least 51", len(records))
	}
	base := append([]byte(nil), seedQWD[:records[50].end]...)
	if got := SHA256(base); got != BaseQWDSHA256 {
		return nil, fmt.Errorf("base QWD checksum %s, want %s", got, BaseQWDSHA256)
	}
	if bytes.Count(base, playerAnchor) != 1 {
		return nil, fmt.Errorf("player anchor occurs %d times, want 1", bytes.Count(base, playerAnchor))
	}
	targetPacketTail := append(append([]byte(nil), playerAnchor...), packetEntitiesAnchor...)
	if bytes.Count(base, targetPacketTail) != 1 {
		return nil, fmt.Errorf("target packet tail occurs %d times, want 1", bytes.Count(base, targetPacketTail))
	}

	scenarios := []Scenario{{Name: "base", Family: "baseline", QWD: base}}
	addUserCommandScenarios(&scenarios, base)
	addPlayerFlagScenarios(&scenarios, base)
	addPacketEntityScenarios(&scenarios, base)
	addServiceScenarios(&scenarios, base)
	addHistoryScenarios(&scenarios, base, records)
	addDemoCommandScenarios(&scenarios, base, records)
	addOuterRecordScenarios(&scenarios, base, records)

	seen := make(map[string]bool, len(scenarios))
	for _, scenario := range scenarios {
		if scenario.Name == "" || scenario.Family == "" || len(scenario.QWD) == 0 {
			return nil, fmt.Errorf("invalid scenario %#v", scenario)
		}
		if seen[scenario.Name] {
			return nil, fmt.Errorf("duplicate scenario %q", scenario.Name)
		}
		seen[scenario.Name] = true
		if _, err := parseRecords(scenario.QWD); err != nil {
			return nil, fmt.Errorf("scenario %s: %w", scenario.Name, err)
		}
	}
	return scenarios, nil
}

func addUserCommandScenarios(scenarios *[]Scenario, base []byte) {
	add := func(name string, mask byte, values map[byte]uint16, packetMsec, commandMsec byte) {
		replacement := playerInfoRecord(uint16(protocol.PFMsec|protocol.PFCommand), mask, values, packetMsec, commandMsec)
		*scenarios = append(*scenarios, Scenario{
			Name: "usercmd/" + name, Family: "usercmd", QWD: replaceInRead(base, playerAnchor, replacement),
		})
	}
	for mask := 0; mask <= math.MaxUint8; mask++ {
		add(fmt.Sprintf("mask_%02x", mask), byte(mask), defaultUserCommandValues(), 0x23, 0x0d)
	}

	wordValues := []uint16{0x0000, 0x0001, 0x007f, 0x0080, 0x7fff, 0x8000, 0xffff}
	byteValues := []uint16{0x0000, 0x0001, 0x007f, 0x0080, 0x00ff}
	for _, field := range userCommandFields() {
		values := wordValues
		if field.size == 1 {
			values = byteValues
		}
		for _, value := range values {
			fieldValues := defaultUserCommandValues()
			fieldValues[field.mask] = value
			add(fmt.Sprintf("%s_%04x", field.name, value), field.mask, fieldValues, 0x23, 0x0d)
		}
	}
	for _, value := range []byte{0x00, 0x01, 0x7f, 0x80, 0xff} {
		add(fmt.Sprintf("packet_msec_%02x", value), protocol.CMAngle1, defaultUserCommandValues(), value, 0x0d)
		add(fmt.Sprintf("command_msec_%02x", value), protocol.CMAngle1, defaultUserCommandValues(), 0x23, value)
	}
}

func addPlayerFlagScenarios(scenarios *[]Scenario, base []byte) {
	supportedFlags := []uint16{
		protocol.PFMsec, protocol.PFCommand,
		protocol.PFVelocity1, protocol.PFVelocity2, protocol.PFVelocity3,
		protocol.PFModel, protocol.PFSkinNum, protocol.PFEffects,
		protocol.PFWeaponFrame, protocol.PFDead,
	}
	for combination := 0; combination < 1<<len(supportedFlags); combination++ {
		var flags uint16
		for i, flag := range supportedFlags {
			if combination&(1<<i) != 0 {
				flags |= flag
			}
		}
		replacement := playerInfoRecord(flags,
			protocol.CMAngle1|protocol.CMAngle2|protocol.CMButtons|protocol.CMImpulse,
			defaultUserCommandValues(), 0x23, 0x0d)
		*scenarios = append(*scenarios, Scenario{
			Name: fmt.Sprintf("player_flags/semantic_%03x", combination), Family: "player_flags",
			QWD: replaceInRead(base, playerAnchor, replacement),
		})
	}

	for gib := 0; gib < 2; gib++ {
		for pmc := 0; pmc < 8; pmc++ {
			for onGround := 0; onGround < 2; onGround++ {
				for solid := 0; solid < 2; solid++ {
					flags := uint16(protocol.PFMsec) | uint16(pmc)<<protocol.PFPMCShift
					if gib != 0 {
						flags |= protocol.PFGib
					}
					if onGround != 0 {
						flags |= protocol.PFOnground
					}
					if solid != 0 {
						flags |= protocol.PFSolid
					}
					name := fmt.Sprintf("player_flags/control_g%d_pmc%d_g%d_s%d", gib, pmc, onGround, solid)
					*scenarios = append(*scenarios, Scenario{
						Name: name, Family: "player_flags",
						QWD: replaceInRead(base, playerAnchor, playerInfoRecord(flags, 0, nil, 0x23, 0)),
					})
				}
			}
		}
	}
}

func addPacketEntityScenarios(scenarios *[]Scenario, base []byte) {
	fields := packetEntityFields()
	for combination := 0; combination < 1<<len(fields); combination++ {
		var mask uint16
		for i, field := range fields {
			if combination&(1<<i) != 0 {
				mask |= field.mask
			}
		}
		payload := packetEntitiesRecord(500, mask, nil)
		*scenarios = append(*scenarios, Scenario{
			Name: fmt.Sprintf("packet_entities/mask_%03x", combination), Family: "packet_entities",
			QWD: replaceTargetPacketEntities(base, payload),
		})
	}

	for _, number := range []uint16{33, 63, 64, 95, 96, 255, 256, 319, 320, 510, 511} {
		payload := packetEntitiesRecord(number, protocol.UModel|protocol.UOrigin1|protocol.UAngle3, nil)
		*scenarios = append(*scenarios, Scenario{
			Name: fmt.Sprintf("packet_entities/number_%03d", number), Family: "packet_entities",
			QWD: replaceTargetPacketEntities(base, payload),
		})
	}
	for _, field := range fields {
		values := []uint16{0, 1, 0x7f, 0x80, 0xff}
		if field.size == 2 {
			values = []uint16{0, 1, 0x7fff, 0x8000, 0xffff}
		}
		for _, value := range values {
			payload := packetEntitiesRecord(500, field.mask, map[uint16]uint16{field.mask: value})
			*scenarios = append(*scenarios, Scenario{
				Name: fmt.Sprintf("packet_entities/%s_%04x", field.name, value), Family: "packet_entities",
				QWD: replaceTargetPacketEntities(base, payload),
			})
		}
	}
}

func addServiceScenarios(scenarios *[]Scenario, base []byte) {
	add := func(name string, payload []byte) {
		*scenarios = append(*scenarios, Scenario{
			Name: "services/" + name, Family: "services",
			QWD: replaceTargetPacketEntities(base, append(append([]byte(nil), payload...), packetEntitiesAnchor...)),
		})
	}
	for _, item := range []struct {
		name   string
		opcode byte
	}{
		{"nop", protocol.SVCNOP}, {"disconnect", protocol.SVCDisconnect},
		{"killed_monster", protocol.SVCKilledMonster}, {"found_secret", protocol.SVCFoundSecret},
		{"sell_screen", protocol.SVCSellScreen}, {"small_kick", protocol.SVCSmallKick},
		{"big_kick", protocol.SVCBigKick},
	} {
		add(item.name, []byte{item.opcode})
	}
	add("update_stat", []byte{protocol.SVCUpdateStat, 0xff, 0x80})
	add("update_frags", []byte{protocol.SVCUpdateFrags, 0x1f, 0xff, 0x7f})
	add("set_angle", []byte{protocol.SVCSetAngle, 0x00, 0x80, 0xff})
	add("update_ping", []byte{protocol.SVCUpdatePing, 0x1f, 0xff, 0x7f})
	spawnStaticSound := append([]byte{protocol.SVCSpawnStaticSound}, words(1, 0x7fff, 0x8000)...)
	spawnStaticSound = append(spawnStaticSound, 1, 0xff, 0x80)
	add("spawn_static_sound", spawnStaticSound)
	intermission := append([]byte{protocol.SVCIntermission}, words(1, 0x7fff, 0x8000)...)
	intermission = append(intermission, 0x00, 0x80, 0xff)
	add("intermission", intermission)
	add("damage_zero", []byte{protocol.SVCDamage, 0xff, 0x80, 0, 0, 0, 0, 0, 0})
	add("damage_position", append([]byte{protocol.SVCDamage, 0xff, 0x80}, words(1, 0x7fff, 0x8000)...))
	add("max_speed", []byte{protocol.SVCMaxSpeed, 0x00, 0x00, 0x80, 0x3f})
	add("entity_gravity", []byte{protocol.SVCEntGravity, 0x00, 0x00, 0x00, 0x3f})
	add("stop_sound", []byte{protocol.SVCStopSound, 0xff, 0x7f})
	add("muzzle_flash", []byte{protocol.SVCMuzzleFlash, 0xff, 0x01})
	add("update_packet_loss", []byte{protocol.SVCUpdatePL, 0x1f, 0xff})
	add("set_pause", []byte{protocol.SVCSetPause, 0x01})
	add("cd_track", []byte{protocol.SVCCDTrack, 0xff})
	add("choke_count", []byte{protocol.SVCChokeCount, 0xff})
	add("update_enter_time", []byte{protocol.SVCUpdateEnterTime, 0x1f, 0x00, 0x00, 0x80, 0x3f})
	add("update_stat_long", []byte{protocol.SVCUpdateStatLong, 0xff, 0x78, 0x56, 0x34, 0x12})
	add("light_style", []byte{protocol.SVCLightStyle, 0xff, 'a', 'b', 'c', 0})
	add("print_low", []byte{protocol.SVCPrint, protocol.PrintLow, 'l', 'o', 'w', 0})
	add("print_medium", []byte{protocol.SVCPrint, protocol.PrintMedium, 'm', 'e', 'd', 0})
	add("print_high", []byte{protocol.SVCPrint, protocol.PrintHigh, 'h', 'i', 0})
	add("print_chat", []byte{protocol.SVCPrint, protocol.PrintChat, 'c', 'h', 'a', 't', 0})
	add("print_high_merge", []byte{
		protocol.SVCPrint, protocol.PrintHigh, 'a', 0,
		protocol.SVCPrint, protocol.PrintHigh, 'b', 0,
	})
	add("stuff_text", []byte{protocol.SVCStuffText, 'c', 'm', 'd', ' ', 'x', '\n', 0})
	add("center_print", []byte{protocol.SVCCenterPrint, 'c', 'e', 'n', 't', 'e', 'r', 0})
	add("set_info", []byte{protocol.SVCSetInfo, 0x1f, 'k', 0, 'v', 0})
	add("server_info", []byte{protocol.SVCServerInfo, 'k', 0, 'v', 0})
	add("update_user_info", []byte{protocol.SVCUpdateUserInfo, 0x1f, 0x78, 0x56, 0x34, 0x12, '\\', 'n', 'a', 'm', 'e', '\\', 'x', 0})
	add("qizmo_string", []byte{qizmo.SVCString, 'q', 'i', 'z', 'm', 'o', 0})
	block := append([]byte{qizmo.SVCBlock}, boundaryBytes(qizmo.SVCBlockPayloadSize)...)
	add("qizmo_block", block)
	voice := append([]byte{qizmo.SVCVoice}, boundaryBytes(qizmo.SVCVoicePayloadSize)...)
	add("qizmo_voice", voice)

	for flags := 0; flags < 4; flags++ {
		channel := uint16(5<<3 | 2)
		if flags&1 != 0 {
			channel |= protocol.SoundVolume
		}
		if flags&2 != 0 {
			channel |= protocol.SoundAttenuation
		}
		payload := []byte{protocol.SVCSound, byte(channel), byte(channel >> 8)}
		if flags&1 != 0 {
			payload = append(payload, 0xff)
		}
		if flags&2 != 0 {
			payload = append(payload, 0x80)
		}
		payload = append(payload, 1)
		payload = append(payload, words(1, 0x7fff, 0x8000)...)
		add(fmt.Sprintf("sound_flags_%d", flags), payload)
	}
	for entityType := byte(0); entityType <= protocol.TELightningBlood; entityType++ {
		payload := []byte{protocol.SVCTempEntity, entityType}
		switch entityType {
		case protocol.TEGunshot, protocol.TEBlood:
			payload = append(payload, 0xff)
			payload = append(payload, words(1, 0x7fff, 0x8000)...)
		case protocol.TELightning1, protocol.TELightning2, protocol.TELightning3:
			payload = append(payload, 0xff, 0x01)
			payload = append(payload, words(1, 2, 3, 0x7fff, 0x8000, 0xffff)...)
		default:
			payload = append(payload, words(1, 0x7fff, 0x8000)...)
		}
		add(fmt.Sprintf("temp_entity_%02d", entityType), payload)
	}
	for _, count := range []byte{0, 1, 2, 16, 255} {
		payload := []byte{protocol.SVCNails, count}
		for i := 0; i < int(count); i++ {
			payload = append(payload, byte(i), byte(i*3), byte(i*5), byte(i*7), byte(i*11), byte(i*13))
		}
		add(fmt.Sprintf("nails_%03d", count), payload)
	}
}

func addHistoryScenarios(scenarios *[]Scenario, base []byte, records []record) {
	throughTarget := append([]byte(nil), base[:records[32].end]...)
	packetStart := records[32].start + qwd.RecordHeaderSize + qwd.ReadSizeFieldSize
	for _, delta := range []uint32{0, 1, 2, 15, 30, 31, 32, 255} {
		qwdData := append([]byte(nil), throughTarget...)
		seq := uint32(11) + delta
		binary.LittleEndian.PutUint32(qwdData[packetStart:], seq)
		binary.LittleEndian.PutUint32(qwdData[packetStart+protocol.QWPacketAckOffset:], seq)
		*scenarios = append(*scenarios, Scenario{
			Name: fmt.Sprintf("history/sequence_delta_%03d", delta), Family: "history", QWD: qwdData,
		})
	}
	for _, bits := range []struct {
		name string
		seq  uint32
		ack  uint32
	}{
		{"ack_mismatch", 12, 11},
		{"sequence_high", 0x8000000c, 12},
		{"ack_high", 12, 0x8000000c},
		{"both_high", 0x8000000c, 0x8000000c},
	} {
		qwdData := append([]byte(nil), throughTarget...)
		binary.LittleEndian.PutUint32(qwdData[packetStart:], bits.seq)
		binary.LittleEndian.PutUint32(qwdData[packetStart+protocol.QWPacketAckOffset:], bits.ack)
		*scenarios = append(*scenarios, Scenario{Name: "history/" + bits.name, Family: "history", QWD: qwdData})
	}
	highMsecPlayer := playerInfoRecord(
		uint16(protocol.PFMsec|protocol.PFCommand),
		protocol.CMAngle1,
		defaultUserCommandValues(),
		0xff,
		0x0d,
	)
	reliableHighMsec := replaceInRead(
		throughTarget,
		playerAnchor,
		highMsecPlayer,
	)
	binary.LittleEndian.PutUint32(reliableHighMsec[packetStart:], 0x8000000c)
	binary.LittleEndian.PutUint32(reliableHighMsec[packetStart+protocol.QWPacketAckOffset:], 0x8000000c)
	*scenarios = append(*scenarios, Scenario{
		Name: "history/reliable_high_msec", Family: "history", QWD: reliableHighMsec,
	})
	reliableLeadingHighMsec := replaceInRead(
		reliableHighMsec,
		highMsecPlayer,
		append([]byte{protocol.SVCNOP}, highMsecPlayer...),
	)
	*scenarios = append(*scenarios, Scenario{
		Name: "history/reliable_leading_high_msec", Family: "history", QWD: reliableLeadingHighMsec,
	})
	withoutEntities := replaceTargetPacketEntities(throughTarget, nil)
	*scenarios = append(*scenarios, Scenario{Name: "history/no_packet_entities", Family: "history", QWD: withoutEntities})
}

func addDemoCommandScenarios(scenarios *[]Scenario, base []byte, records []record) {
	const targetRecord = 35
	original := base[records[targetRecord].start:records[targetRecord].end]
	if records[targetRecord].command != protocol.DemoCmd ||
		len(original) != qwd.RecordHeaderSize+qwd.CmdPayloadSize {
		panic("unexpected demo command anchor")
	}
	add := func(name string, payload [qwd.CmdPayloadSize]byte) {
		qwdData := append([]byte(nil), base...)
		copy(
			qwdData[records[targetRecord].start+qwd.RecordHeaderSize:records[targetRecord].end],
			payload[:],
		)
		*scenarios = append(*scenarios, Scenario{Name: "demo_cmd/" + name, Family: "demo_cmd", QWD: qwdData})
	}
	for axis := 0; axis < 3; axis++ {
		for _, value := range []int16{math.MinInt16, -1, 0, 1, math.MaxInt16} {
			angles := [3]int16{}
			angles[axis] = value
			add(fmt.Sprintf("angle%d_%04x", axis, uint16(value)), makeDemoCommand(13, angles, [3]int16{}, 0, 8))
		}
	}
	for axis := 0; axis < 3; axis++ {
		for _, value := range []int16{math.MinInt16, -1, 0, 1, math.MaxInt16} {
			movement := [3]int16{}
			movement[axis] = value
			add(fmt.Sprintf("move%d_%04x", axis, uint16(value)), makeDemoCommand(13, [3]int16{}, movement, 0, 8))
		}
	}
	for _, value := range []byte{0, 1, 0x7f, 0x80, 0xff} {
		add(fmt.Sprintf("buttons_%02x", value), makeDemoCommand(13, [3]int16{}, [3]int16{}, value, 8))
	}
	for _, value := range []byte{1, 8, 0x7f, 0x80, 0xff} {
		add(fmt.Sprintf("impulse_%02x", value), makeDemoCommand(13, [3]int16{}, [3]int16{}, 0, value))
	}
	for _, value := range []byte{0, 1, 0x7f, 0x80, 0xff} {
		add(fmt.Sprintf("msec_%02x", value), makeDemoCommand(value, [3]int16{}, [3]int16{}, 0, 8))
	}
}

func addOuterRecordScenarios(scenarios *[]Scenario, base []byte, records []record) {
	for _, value := range []uint32{0, 1, 0x7fffffff, 0x80000000, 0xffffffff} {
		setRecord := (&qwd.Data{
			Command: protocol.DemoSet,
			Set:     &qwd.Set{SeqOut: value, SeqIn: ^value},
		}).Bytes()
		qwdData := append([]byte(nil), setRecord...)
		qwdData = append(qwdData, base...)
		*scenarios = append(*scenarios, Scenario{
			Name: fmt.Sprintf("outer/set_%08x", value), Family: "outer", QWD: qwdData,
		})
	}

	addTimestamp := func(name string, lastRecord int, timestampRecord int, timestamp float32) {
		qwdData := append([]byte(nil), base[:records[lastRecord].end]...)
		binary.LittleEndian.PutUint32(
			qwdData[records[timestampRecord].start:],
			math.Float32bits(timestamp),
		)
		*scenarios = append(*scenarios, Scenario{
			Name: "outer/timestamp_" + name, Family: "outer", QWD: qwdData,
		})
	}
	addTimestamp("initial_ignored", 2, 0, 12.345)
	addTimestamp("without_command_ignored", 33, 33, 9.5)
	addTimestamp("fractional_rounded", 36, 36, 0.0834)
	addTimestamp("large_advance_clamped", 36, 36, 1.070)
	addTimestamp("backward_clamped", 36, 36, 0.050)
}

type commandField struct {
	name string
	mask byte
	size int
}

func userCommandFields() []commandField {
	return []commandField{
		{"angle1", protocol.CMAngle1, 2},
		{"angle2", protocol.CMAngle2, 2},
		{"angle3", protocol.CMAngle3, 2},
		{"forward", protocol.CMForward, 2},
		{"side", protocol.CMSide, 2},
		{"up", protocol.CMUp, 2},
		{"buttons", protocol.CMButtons, 1},
		{"impulse", protocol.CMImpulse, 1},
	}
}

func defaultUserCommandValues() map[byte]uint16 {
	return map[byte]uint16{
		protocol.CMAngle1: 0x068b, protocol.CMAngle2: 0x3979,
		protocol.CMAngle3: 0x3333, protocol.CMForward: 0x4444,
		protocol.CMSide: 0x5555, protocol.CMUp: 0x6666,
		protocol.CMButtons: 0x0077, protocol.CMImpulse: 0x0088,
	}
}

func playerInfoRecord(flags uint16, extra byte, values map[byte]uint16, packetMsec, commandMsec byte) []byte {
	out := []byte{protocol.SVCPlayerInfo, 1, byte(flags), byte(flags >> 8), 0x19, 0x18, 0xe6, 0xf9, 0x40, 0x03, 0x0e}
	if flags&protocol.PFMsec != 0 {
		out = append(out, packetMsec)
	}
	if flags&protocol.PFCommand != 0 {
		out = append(out, extra)
		for _, field := range userCommandFields() {
			if extra&field.mask == 0 {
				continue
			}
			value := values[field.mask]
			out = append(out, byte(value))
			if field.size == 2 {
				out = append(out, byte(value>>8))
			}
		}
		out = append(out, commandMsec)
	}
	for _, field := range []struct {
		mask  uint16
		value uint16
	}{
		{protocol.PFVelocity1, 0x1111}, {protocol.PFVelocity2, 0x2222}, {protocol.PFVelocity3, 0x3333},
	} {
		if flags&field.mask != 0 {
			out = binary.LittleEndian.AppendUint16(out, field.value)
		}
	}
	for _, field := range []struct {
		mask  uint16
		value byte
	}{
		{protocol.PFModel, 2}, {protocol.PFSkinNum, 3}, {protocol.PFEffects, 4}, {protocol.PFWeaponFrame, 5},
	} {
		if flags&field.mask != 0 {
			out = append(out, field.value)
		}
	}
	return out
}

type entityField struct {
	name  string
	mask  uint16
	size  int
	value uint16
}

func packetEntityFields() []entityField {
	return []entityField{
		{"model", protocol.UModel, 1, 2}, {"frame", protocol.UFrame, 1, 3},
		{"colormap", protocol.UColorMap, 1, 4}, {"skin", protocol.USkin, 1, 5},
		{"effects", protocol.UEffects, 1, 6}, {"origin1", protocol.UOrigin1, 2, 0x1111},
		{"angle1", protocol.UAngle1, 1, 0x22}, {"origin2", protocol.UOrigin2, 2, 0x3333},
		{"angle2", protocol.UAngle2, 1, 0x44}, {"origin3", protocol.UOrigin3, 2, 0x5555},
		{"angle3", protocol.UAngle3, 1, 0x66},
	}
}

func packetEntitiesRecord(number, mask uint16, overrides map[uint16]uint16) []byte {
	wireBits := mask
	if wireBits&0x00ff != 0 {
		wireBits |= protocol.UMoreBits
	}
	header := number | wireBits
	out := []byte{protocol.SVCPacketEntities, byte(header), byte(header >> 8)}
	if wireBits&protocol.UMoreBits != 0 {
		out = append(out, byte(wireBits))
	}
	for _, field := range packetEntityFields() {
		if mask&field.mask == 0 {
			continue
		}
		value := field.value
		if override, ok := overrides[field.mask]; ok {
			value = override
		}
		out = append(out, byte(value))
		if field.size == 2 {
			out = append(out, byte(value>>8))
		}
	}
	return append(out, 0, 0)
}

func makeDemoCommand(msec byte, angles [3]int16, movement [3]int16, buttons, impulse byte) [qwd.CmdPayloadSize]byte {
	command := qwd.Cmd{
		Msec:    msec,
		Forward: uint16(movement[0]),
		Side:    uint16(movement[1]),
		Up:      uint16(movement[2]),
		Buttons: buttons,
		Impulse: impulse,
	}
	for i, angle := range angles {
		value := float32(float64(angle) * (360.0 / 65536.0))
		command.UserAngle[i] = value
		command.Angle[i] = value
	}

	var payload [qwd.CmdPayloadSize]byte
	copy(payload[:], command.Bytes())
	return payload
}

func replaceInRead(qwdData, old, replacement []byte) []byte {
	index := bytes.Index(qwdData, old)
	if index < 0 || bytes.Contains(qwdData[index+len(old):], old) {
		panic(fmt.Sprintf("anchor occurrence for %x is not unique", old[:min(len(old), 8)]))
	}
	records, err := parseRecords(qwdData)
	if err != nil {
		panic(err)
	}
	for _, record := range records {
		payloadStart := record.start + qwd.RecordHeaderSize + qwd.ReadSizeFieldSize
		if record.command != protocol.DemoRead || index < payloadStart || index+len(old) > record.end {
			continue
		}
		result := make([]byte, 0, len(qwdData)-len(old)+len(replacement))
		result = append(result, qwdData[:index]...)
		result = append(result, replacement...)
		result = append(result, qwdData[index+len(old):]...)
		sizeOffset := record.start + qwd.RecordHeaderSize
		size := int(binary.LittleEndian.Uint32(
			qwdData[sizeOffset : sizeOffset+qwd.ReadSizeFieldSize],
		))
		size += len(replacement) - len(old)
		binary.LittleEndian.PutUint32(result[sizeOffset:sizeOffset+qwd.ReadSizeFieldSize], uint32(size))
		return result
	}
	panic("anchor is not inside a DEMO_READ record")
}

func replaceTargetPacketEntities(qwdData, replacement []byte) []byte {
	old := append(append([]byte(nil), playerAnchor...), packetEntitiesAnchor...)
	next := append(append([]byte(nil), playerAnchor...), replacement...)
	return replaceInRead(qwdData, old, next)
}

func parseRecords(data []byte) ([]record, error) {
	var records []record
	for off := 0; off < len(data); {
		start := off
		if len(data)-off < qwd.RecordHeaderSize {
			return nil, fmt.Errorf("record %d truncated header", len(records))
		}
		command := data[off+qwd.TimestampSize]
		off += qwd.RecordHeaderSize
		switch command {
		case protocol.DemoCmd:
			off += qwd.CmdPayloadSize
		case protocol.DemoRead:
			if len(data)-off < qwd.ReadSizeFieldSize {
				return nil, fmt.Errorf("record %d truncated size", len(records))
			}
			size := int(binary.LittleEndian.Uint32(data[off : off+qwd.ReadSizeFieldSize]))
			off += qwd.ReadSizeFieldSize + size
		case protocol.DemoSet:
			off += qwd.SetPayloadSize
		default:
			return nil, fmt.Errorf("record %d unknown command %d", len(records), command)
		}
		if off < 0 || off > len(data) {
			return nil, fmt.Errorf("record %d extends to %d beyond %d", len(records), off, len(data))
		}
		records = append(records, record{start: start, end: off, command: command})
	}
	return records, nil
}

func words(values ...uint16) []byte {
	out := make([]byte, 0, len(values)*2)
	for _, value := range values {
		out = binary.LittleEndian.AppendUint16(out, value)
	}
	return out
}

func boundaryBytes(size int) []byte {
	values := make([]byte, size)
	boundaries := []byte{0x00, 0x01, 0x7f, 0x80, 0xfe, 0xff}
	for i := range values {
		values[i] = boundaries[i%len(boundaries)]
	}
	return values
}

func SHA256(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}
