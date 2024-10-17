package svc

import (
	"errors"

	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/bad"
	"github.com/osm/quake/packet/command/bigkick"
	"github.com/osm/quake/packet/command/cdtrack"
	"github.com/osm/quake/packet/command/centerprint"
	"github.com/osm/quake/packet/command/chokecount"
	"github.com/osm/quake/packet/command/clientdata"
	"github.com/osm/quake/packet/command/damage"
	"github.com/osm/quake/packet/command/deltapacketentities"
	"github.com/osm/quake/packet/command/disconnect"
	"github.com/osm/quake/packet/command/download"
	"github.com/osm/quake/packet/command/entgravity"
	"github.com/osm/quake/packet/command/fastupdate"
	"github.com/osm/quake/packet/command/finale"
	"github.com/osm/quake/packet/command/foundsecret"
	"github.com/osm/quake/packet/command/ftemodellist"
	"github.com/osm/quake/packet/command/ftespawnbaseline"
	"github.com/osm/quake/packet/command/ftespawnstatic"
	"github.com/osm/quake/packet/command/ftevoicechats"
	"github.com/osm/quake/packet/command/intermission"
	"github.com/osm/quake/packet/command/killedmonster"
	"github.com/osm/quake/packet/command/lightstyle"
	"github.com/osm/quake/packet/command/maxspeed"
	"github.com/osm/quake/packet/command/modellist"
	"github.com/osm/quake/packet/command/muzzleflash"
	"github.com/osm/quake/packet/command/nails"
	"github.com/osm/quake/packet/command/nails2"
	"github.com/osm/quake/packet/command/nops"
	"github.com/osm/quake/packet/command/packetentities"
	"github.com/osm/quake/packet/command/particle"
	"github.com/osm/quake/packet/command/playerinfo"
	"github.com/osm/quake/packet/command/print"
	"github.com/osm/quake/packet/command/qizmovoice"
	"github.com/osm/quake/packet/command/sellscreen"
	"github.com/osm/quake/packet/command/serverdata"
	"github.com/osm/quake/packet/command/serverinfo"
	"github.com/osm/quake/packet/command/setangle"
	"github.com/osm/quake/packet/command/setinfo"
	"github.com/osm/quake/packet/command/setpause"
	"github.com/osm/quake/packet/command/setview"
	"github.com/osm/quake/packet/command/signonnum"
	"github.com/osm/quake/packet/command/smallkick"
	"github.com/osm/quake/packet/command/sound"
	"github.com/osm/quake/packet/command/soundlist"
	"github.com/osm/quake/packet/command/spawnbaseline"
	"github.com/osm/quake/packet/command/spawnstatic"
	"github.com/osm/quake/packet/command/spawnstaticsound"
	"github.com/osm/quake/packet/command/stopsound"
	"github.com/osm/quake/packet/command/stufftext"
	"github.com/osm/quake/packet/command/tempentity"
	"github.com/osm/quake/packet/command/time"
	"github.com/osm/quake/packet/command/updatecolors"
	"github.com/osm/quake/packet/command/updateentertime"
	"github.com/osm/quake/packet/command/updatefrags"
	"github.com/osm/quake/packet/command/updatename"
	"github.com/osm/quake/packet/command/updateping"
	"github.com/osm/quake/packet/command/updatepl"
	"github.com/osm/quake/packet/command/updatestat"
	"github.com/osm/quake/packet/command/updatestatlong"
	"github.com/osm/quake/packet/command/updateuserinfo"
	"github.com/osm/quake/packet/command/version"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/protocol/fte"
)

var ErrUnknownCommandType = errors.New("unknown command type")

type GameData struct {
	IsMVD bool
	IsNQ  bool

	Seq      uint32
	Ack      uint32
	Commands []command.Command
}

func (gd *GameData) Bytes() []byte {
	buf := buffer.New()

	if gd.IsMVD || gd.IsNQ {
		goto process
	}

	buf.PutUint32(gd.Seq)
	buf.PutUint32(gd.Ack)

process:
	for _, c := range gd.Commands {
		buf.PutBytes(c.Bytes())
	}

	return buf.Bytes()
}

func parseGameData(ctx *context.Context, buf *buffer.Buffer) (*GameData, error) {
	var err error
	var pkg GameData

	pkg.IsMVD = ctx.GetIsMVD()
	pkg.IsNQ = ctx.GetIsNQ()

	if pkg.IsMVD || pkg.IsNQ {
		goto process
	}

	if pkg.Seq, err = buf.GetUint32(); err != nil {
		return nil, err
	}

	if pkg.Ack, err = buf.GetUint32(); err != nil {
		return nil, err
	}

process:
	var cmd command.Command
	for buf.Off() < buf.Len() {
		typ, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}

		if pkg.IsNQ && typ&128 != 0 {
			cmd, err = fastupdate.Parse(ctx, buf, typ)
			goto next
		}

		switch protocol.CommandType(typ) {
		case protocol.SVCBad:
			cmd, err = bad.Parse(ctx, buf, protocol.SVCBad)
		case protocol.SVCNOP:
			cmd, err = nops.Parse(ctx, buf)
		case protocol.SVCDisconnect:
			cmd, err = disconnect.Parse(ctx, buf)
		case protocol.SVCUpdateStat:
			cmd, err = updatestat.Parse(ctx, buf)
		case protocol.SVCVersion:
			cmd, err = version.Parse(ctx, buf)
		case protocol.SVCSetView:
			cmd, err = setview.Parse(ctx, buf)
		case protocol.SVCSound:
			cmd, err = sound.Parse(ctx, buf)
		case protocol.SVCTime:
			cmd, err = time.Parse(ctx, buf)
		case protocol.SVCPrint:
			cmd, err = print.Parse(ctx, buf)
		case protocol.SVCStuffText:
			cmd, err = stufftext.Parse(ctx, buf)
		case protocol.SVCSetAngle:
			cmd, err = setangle.Parse(ctx, buf)
		case protocol.SVCServerData:
			cmd, err = serverdata.Parse(ctx, buf)
		case protocol.SVCLightStyle:
			cmd, err = lightstyle.Parse(ctx, buf)
		case protocol.SVCUpdateName:
			cmd, err = updatename.Parse(ctx, buf)
		case protocol.SVCUpdateFrags:
			cmd, err = updatefrags.Parse(ctx, buf)
		case protocol.SVCClientData:
			cmd, err = clientdata.Parse(ctx, buf)
		case protocol.SVCStopSound:
			cmd, err = stopsound.Parse(ctx, buf)
		case protocol.SVCUpdateColors:
			cmd, err = updatecolors.Parse(ctx, buf)
		case protocol.SVCParticle:
			cmd, err = particle.Parse(ctx, buf)
		case protocol.SVCDamage:
			cmd, err = damage.Parse(ctx, buf)
		case protocol.SVCSpawnStatic:
			cmd, err = spawnstatic.Parse(ctx, buf)
		case protocol.SVCSpawnBaseline:
			cmd, err = spawnbaseline.Parse(ctx, buf)
		case protocol.SVCTempEntity:
			cmd, err = tempentity.Parse(ctx, buf)
		case protocol.SVCSetPause:
			cmd, err = setpause.Parse(ctx, buf)
		case protocol.SVCSignOnNum:
			cmd, err = signonnum.Parse(ctx, buf)
		case protocol.SVCCenterPrint:
			cmd, err = centerprint.Parse(ctx, buf)
		case protocol.SVCKilledMonster:
			cmd, err = killedmonster.Parse(ctx, buf)
		case protocol.SVCFoundSecret:
			cmd, err = foundsecret.Parse(ctx, buf)
		case protocol.SVCSpawnStaticSound:
			cmd, err = spawnstaticsound.Parse(ctx, buf)
		case protocol.SVCIntermission:
			cmd, err = intermission.Parse(ctx, buf)
		case protocol.SVCFinale:
			cmd, err = finale.Parse(ctx, buf)
		case protocol.SVCCDTrack:
			cmd, err = cdtrack.Parse(ctx, buf)
		case protocol.SVCSellScreen:
			cmd, err = sellscreen.Parse(ctx, buf)
		case protocol.SVCSmallKick:
			cmd, err = smallkick.Parse(ctx, buf)
		case protocol.SVCBigKick:
			cmd, err = bigkick.Parse(ctx, buf)
		case protocol.SVCUpdatePing:
			cmd, err = updateping.Parse(ctx, buf)
		case protocol.SVCUpdateEnterTime:
			cmd, err = updateentertime.Parse(ctx, buf)
		case protocol.SVCUpdateStatLong:
			cmd, err = updatestatlong.Parse(ctx, buf)
		case protocol.SVCMuzzleFlash:
			cmd, err = muzzleflash.Parse(ctx, buf)
		case protocol.SVCUpdateUserInfo:
			cmd, err = updateuserinfo.Parse(ctx, buf)
		case protocol.SVCDownload:
			cmd, err = download.Parse(ctx, buf)
		case protocol.SVCPlayerInfo:
			cmd, err = playerinfo.Parse(ctx, buf)
		case protocol.SVCNails:
			cmd, err = nails.Parse(ctx, buf)
		case protocol.SVCChokeCount:
			cmd, err = chokecount.Parse(ctx, buf)
		case protocol.SVCModelList:
			cmd, err = modellist.Parse(ctx, buf)
		case protocol.SVCSoundList:
			cmd, err = soundlist.Parse(ctx, buf)
		case protocol.SVCPacketEntities:
			cmd, err = packetentities.Parse(ctx, buf)
		case protocol.SVCDeltaPacketEntities:
			cmd, err = deltapacketentities.Parse(ctx, buf)
		case protocol.SVCMaxSpeed:
			cmd, err = maxspeed.Parse(ctx, buf)
		case protocol.SVCEntGravity:
			cmd, err = entgravity.Parse(ctx, buf)
		case protocol.SVCSetInfo:
			cmd, err = setinfo.Parse(ctx, buf)
		case protocol.SVCServerInfo:
			cmd, err = serverinfo.Parse(ctx, buf)
		case protocol.SVCUpdatePL:
			cmd, err = updatepl.Parse(ctx, buf)
		case protocol.SVCNails2:
			cmd, err = nails2.Parse(ctx, buf)
		case protocol.SVCQizmoVoice:
			cmd, err = qizmovoice.Parse(ctx, buf)
		case fte.SVCSpawnStatic:
			cmd, err = ftespawnstatic.Parse(ctx, buf)
		case fte.SVCModelListShort:
			cmd, err = ftemodellist.Parse(ctx, buf)
		case fte.SVCSpawnBaseline:
			cmd, err = ftespawnbaseline.Parse(ctx, buf)
		case fte.SVCVoiceChat:
			cmd, err = ftevoicechats.Parse(ctx, buf)
		default:
			return nil, ErrUnknownCommandType
		}

	next:
		if err != nil {
			return nil, err
		}
		pkg.Commands = append(pkg.Commands, cmd)
	}

	return &pkg, nil
}
