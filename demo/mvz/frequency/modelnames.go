package frequency

import "fmt"

func (model Model) String() string {
	if model < ModelCount && modelNames[model] != "" {
		return modelNames[model]
	}
	return fmt.Sprintf("model_%d", model)
}

// Report labels are keyed by model ID. They are not stored in MVZ demos
// or MVZF frequency artifacts.
var modelNames = [ModelCount]string{
	// MVD record fields.
	ModelRawByte:        "raw_byte",
	ModelRecordKind:     "record_kind",
	ModelTimestamp:      "timestamp",
	ModelCommandTarget:  "command_target",
	ModelLength:         "length",
	ModelRecordBodyMode: "record_body_mode",
	ModelOperationKind:  "operation_kind",

	// Player fields.
	ModelPlayerCoordSize:        "player_coord_size",
	ModelPlayerIndex:            "player_index",
	ModelPlayerBitsLoXOR:        "player_bits_lo_xor",
	ModelPlayerBitsHiXOR:        "player_bits_hi_xor",
	ModelPlayerFrameDelta:       "player_frame_delta",
	ModelPlayerOriginXLoDelta:   "player_origin_x_lo_delta",
	ModelPlayerOriginXHiDelta:   "player_origin_x_hi_delta",
	ModelPlayerOriginYLoDelta:   "player_origin_y_lo_delta",
	ModelPlayerOriginYHiDelta:   "player_origin_y_hi_delta",
	ModelPlayerOriginZLoDelta:   "player_origin_z_lo_delta",
	ModelPlayerOriginZHiDelta:   "player_origin_z_hi_delta",
	ModelPlayerOriginByte2Delta: "player_origin_byte2_delta",
	ModelPlayerOriginByte3Delta: "player_origin_byte3_delta",
	ModelPlayerPitchLoDelta:     "player_pitch_lo_delta",
	ModelPlayerPitchHiDelta:     "player_pitch_hi_delta",
	ModelPlayerYawLoDelta:       "player_yaw_lo_delta",
	ModelPlayerYawHiDelta:       "player_yaw_hi_delta",
	ModelPlayerRollLoDelta:      "player_roll_lo_delta",
	ModelPlayerRollHiDelta:      "player_roll_hi_delta",
	ModelPlayerModelDelta:       "player_model_delta",
	ModelPlayerSkinDelta:        "player_skin_delta",
	ModelPlayerEffectsXOR:       "player_effects_xor",
	ModelPlayerWeaponFrameDelta: "player_weapon_frame_delta",

	// Packet entity fields.
	ModelEntityDeltaBase:        "entity_delta_base",
	ModelEntityNumberLoDelta:    "entity_number_lo_delta",
	ModelEntityNumberHiDelta:    "entity_number_hi_delta",
	ModelEntityBitsXOR:          "entity_bits_xor",
	ModelEntityMoreBitsXOR:      "entity_more_bits_xor",
	ModelEntityEvenMoreBitsXOR:  "entity_even_more_bits_xor",
	ModelEntityYetMoreBitsXOR:   "entity_yet_more_bits_xor",
	ModelEntityCoordSize:        "entity_coord_size",
	ModelEntityAngleSize:        "entity_angle_size",
	ModelEntityModelDelta:       "entity_model_delta",
	ModelEntityFrameDelta:       "entity_frame_delta",
	ModelEntityColorMapDelta:    "entity_colormap_delta",
	ModelEntitySkinDelta:        "entity_skin_delta",
	ModelEntityEffectsXOR:       "entity_effects_xor",
	ModelEntityOriginXLoDelta:   "entity_origin_x_lo_delta",
	ModelEntityOriginXHiDelta:   "entity_origin_x_hi_delta",
	ModelEntityOriginYLoDelta:   "entity_origin_y_lo_delta",
	ModelEntityOriginYHiDelta:   "entity_origin_y_hi_delta",
	ModelEntityOriginZLoDelta:   "entity_origin_z_lo_delta",
	ModelEntityOriginZHiDelta:   "entity_origin_z_hi_delta",
	ModelEntityOriginByte2Delta: "entity_origin_byte2_delta",
	ModelEntityOriginByte3Delta: "entity_origin_byte3_delta",
	ModelEntityAngle1LoDelta:    "entity_angle1_lo_delta",
	ModelEntityAngle1HiDelta:    "entity_angle1_hi_delta",
	ModelEntityAngle2LoDelta:    "entity_angle2_lo_delta",
	ModelEntityAngle2HiDelta:    "entity_angle2_hi_delta",
	ModelEntityAngle3LoDelta:    "entity_angle3_lo_delta",
	ModelEntityAngle3HiDelta:    "entity_angle3_hi_delta",
	ModelEntityTailByte:         "entity_tail_byte",

	// Record and tail lengths.
	ModelRecordLengthSet:      "record_length_set",
	ModelRecordLengthCommand:  "record_length_command",
	ModelRecordLengthMultiple: "record_length_multiple",
	ModelRecordLengthSingle:   "record_length_single",
	ModelRecordLengthStats:    "record_length_stats",
	ModelRecordLengthAll:      "record_length_all",
	ModelTrailingLength:       "trailing_length",
	ModelEntityTailLength:     "entity_tail_length",

	// Player angles: previous prediction residual magnitude.
	ModelPlayerPitchLoDeltaPrevSmall:  "player_pitch_lo_delta_prev_small",
	ModelPlayerPitchHiDeltaPrevSmall:  "player_pitch_hi_delta_prev_small",
	ModelPlayerPitchLoDeltaPrevMedium: "player_pitch_lo_delta_prev_medium",
	ModelPlayerPitchHiDeltaPrevMedium: "player_pitch_hi_delta_prev_medium",
	ModelPlayerPitchLoDeltaPrevLarge:  "player_pitch_lo_delta_prev_large",
	ModelPlayerPitchHiDeltaPrevLarge:  "player_pitch_hi_delta_prev_large",
	ModelPlayerYawLoDeltaPrevSmall:    "player_yaw_lo_delta_prev_small",
	ModelPlayerYawHiDeltaPrevSmall:    "player_yaw_hi_delta_prev_small",
	ModelPlayerYawLoDeltaPrevMedium:   "player_yaw_lo_delta_prev_medium",
	ModelPlayerYawHiDeltaPrevMedium:   "player_yaw_hi_delta_prev_medium",
	ModelPlayerYawLoDeltaPrevLarge:    "player_yaw_lo_delta_prev_large",
	ModelPlayerYawHiDeltaPrevLarge:    "player_yaw_hi_delta_prev_large",

	// Player origins: previous prediction residual magnitude.
	ModelPlayerOriginXLoDeltaPrevSmall:  "player_origin_x_lo_delta_prev_small",
	ModelPlayerOriginXHiDeltaPrevSmall:  "player_origin_x_hi_delta_prev_small",
	ModelPlayerOriginXLoDeltaPrevMedium: "player_origin_x_lo_delta_prev_medium",
	ModelPlayerOriginXHiDeltaPrevMedium: "player_origin_x_hi_delta_prev_medium",
	ModelPlayerOriginXLoDeltaPrevLarge:  "player_origin_x_lo_delta_prev_large",
	ModelPlayerOriginXHiDeltaPrevLarge:  "player_origin_x_hi_delta_prev_large",
	ModelPlayerOriginYLoDeltaPrevSmall:  "player_origin_y_lo_delta_prev_small",
	ModelPlayerOriginYHiDeltaPrevSmall:  "player_origin_y_hi_delta_prev_small",
	ModelPlayerOriginYLoDeltaPrevMedium: "player_origin_y_lo_delta_prev_medium",
	ModelPlayerOriginYHiDeltaPrevMedium: "player_origin_y_hi_delta_prev_medium",
	ModelPlayerOriginYLoDeltaPrevLarge:  "player_origin_y_lo_delta_prev_large",
	ModelPlayerOriginYHiDeltaPrevLarge:  "player_origin_y_hi_delta_prev_large",
	ModelPlayerOriginZLoDeltaPrevSmall:  "player_origin_z_lo_delta_prev_small",
	ModelPlayerOriginZHiDeltaPrevSmall:  "player_origin_z_hi_delta_prev_small",
	ModelPlayerOriginZLoDeltaPrevMedium: "player_origin_z_lo_delta_prev_medium",
	ModelPlayerOriginZHiDeltaPrevMedium: "player_origin_z_hi_delta_prev_medium",
	ModelPlayerOriginZLoDeltaPrevLarge:  "player_origin_z_lo_delta_prev_large",
	ModelPlayerOriginZHiDeltaPrevLarge:  "player_origin_z_hi_delta_prev_large",

	// Entity numbers: previous delta magnitude.
	ModelEntityNumberLoDeltaPrevSmall:  "entity_number_lo_delta_prev_small",
	ModelEntityNumberHiDeltaPrevSmall:  "entity_number_hi_delta_prev_small",
	ModelEntityNumberLoDeltaPrevMedium: "entity_number_lo_delta_prev_medium",
	ModelEntityNumberHiDeltaPrevMedium: "entity_number_hi_delta_prev_medium",
	ModelEntityNumberLoDeltaPrevLarge:  "entity_number_lo_delta_prev_large",
	ModelEntityNumberHiDeltaPrevLarge:  "entity_number_hi_delta_prev_large",

	// Entity origins: previous prediction residual magnitude.
	ModelEntityOriginXLoDeltaPrevSmall:  "entity_origin_x_lo_delta_prev_small",
	ModelEntityOriginXHiDeltaPrevSmall:  "entity_origin_x_hi_delta_prev_small",
	ModelEntityOriginXLoDeltaPrevMedium: "entity_origin_x_lo_delta_prev_medium",
	ModelEntityOriginXHiDeltaPrevMedium: "entity_origin_x_hi_delta_prev_medium",
	ModelEntityOriginXLoDeltaPrevLarge:  "entity_origin_x_lo_delta_prev_large",
	ModelEntityOriginXHiDeltaPrevLarge:  "entity_origin_x_hi_delta_prev_large",
	ModelEntityOriginYLoDeltaPrevSmall:  "entity_origin_y_lo_delta_prev_small",
	ModelEntityOriginYHiDeltaPrevSmall:  "entity_origin_y_hi_delta_prev_small",
	ModelEntityOriginYLoDeltaPrevMedium: "entity_origin_y_lo_delta_prev_medium",
	ModelEntityOriginYHiDeltaPrevMedium: "entity_origin_y_hi_delta_prev_medium",
	ModelEntityOriginYLoDeltaPrevLarge:  "entity_origin_y_lo_delta_prev_large",
	ModelEntityOriginYHiDeltaPrevLarge:  "entity_origin_y_hi_delta_prev_large",
	ModelEntityOriginZLoDeltaPrevSmall:  "entity_origin_z_lo_delta_prev_small",
	ModelEntityOriginZHiDeltaPrevSmall:  "entity_origin_z_hi_delta_prev_small",
	ModelEntityOriginZLoDeltaPrevMedium: "entity_origin_z_lo_delta_prev_medium",
	ModelEntityOriginZHiDeltaPrevMedium: "entity_origin_z_hi_delta_prev_medium",
	ModelEntityOriginZLoDeltaPrevLarge:  "entity_origin_z_lo_delta_prev_large",
	ModelEntityOriginZHiDeltaPrevLarge:  "entity_origin_z_hi_delta_prev_large",

	// Previous operation context.
	ModelOperationKindAfterRaw:    "operation_kind_after_raw",
	ModelOperationKindAfterPlayer: "operation_kind_after_player",
	ModelOperationKindAfterPacket: "operation_kind_after_packet",
	ModelOperationKindAfterDelta:  "operation_kind_after_delta",

	// Record kind and body mode context.
	ModelRecordKindAfter0:    "record_kind_after_0",
	ModelRecordKindAfter1:    "record_kind_after_1",
	ModelRecordKindAfter2:    "record_kind_after_2",
	ModelRecordKindAfter3:    "record_kind_after_3",
	ModelRecordKindAfter4:    "record_kind_after_4",
	ModelRecordKindAfter5:    "record_kind_after_5",
	ModelRecordKindAfter6:    "record_kind_after_6",
	ModelRecordBodyModeKind1: "record_body_mode_kind_1",
	ModelRecordBodyModeKind2: "record_body_mode_kind_2",
	ModelRecordBodyModeKind3: "record_body_mode_kind_3",
	ModelRecordBodyModeKind4: "record_body_mode_kind_4",
	ModelRecordBodyModeKind5: "record_body_mode_kind_5",
	ModelRecordBodyModeKind6: "record_body_mode_kind_6",

	// Timestamps and destinations by record kind.
	ModelTimestampKind1:     "timestamp_kind_1",
	ModelTimestampKind2:     "timestamp_kind_2",
	ModelTimestampKind3:     "timestamp_kind_3",
	ModelTimestampKind4:     "timestamp_kind_4",
	ModelTimestampKind5:     "timestamp_kind_5",
	ModelTimestampKind6:     "timestamp_kind_6",
	ModelCommandTargetKind1: "command_target_kind_1",
	ModelCommandTargetKind2: "command_target_kind_2",
	ModelCommandTargetKind3: "command_target_kind_3",
	ModelCommandTargetKind4: "command_target_kind_4",
	ModelCommandTargetKind5: "command_target_kind_5",
	ModelCommandTargetKind6: "command_target_kind_6",

	// Player index and nonzero timestamp context.
	ModelPlayerIndexPrevSmall:      "player_index_prev_small",
	ModelPlayerIndexPrevMedium:     "player_index_prev_medium",
	ModelPlayerIndexPrevLarge:      "player_index_prev_large",
	ModelTimestampKind0PrevNonzero: "timestamp_kind_0_prev_nonzero",
	ModelTimestampKind1PrevNonzero: "timestamp_kind_1_prev_nonzero",
	ModelTimestampKind2PrevNonzero: "timestamp_kind_2_prev_nonzero",
	ModelTimestampKind3PrevNonzero: "timestamp_kind_3_prev_nonzero",
	ModelTimestampKind4PrevNonzero: "timestamp_kind_4_prev_nonzero",
	ModelTimestampKind5PrevNonzero: "timestamp_kind_5_prev_nonzero",
	ModelTimestampKind6PrevNonzero: "timestamp_kind_6_prev_nonzero",

	// Entity angles: previous prediction residual magnitude.
	ModelEntityAngle1LoDeltaPrevSmall:  "entity_angle1_lo_delta_prev_small",
	ModelEntityAngle1HiDeltaPrevSmall:  "entity_angle1_hi_delta_prev_small",
	ModelEntityAngle1LoDeltaPrevMedium: "entity_angle1_lo_delta_prev_medium",
	ModelEntityAngle1HiDeltaPrevMedium: "entity_angle1_hi_delta_prev_medium",
	ModelEntityAngle1LoDeltaPrevLarge:  "entity_angle1_lo_delta_prev_large",
	ModelEntityAngle1HiDeltaPrevLarge:  "entity_angle1_hi_delta_prev_large",
	ModelEntityAngle2LoDeltaPrevSmall:  "entity_angle2_lo_delta_prev_small",
	ModelEntityAngle2HiDeltaPrevSmall:  "entity_angle2_hi_delta_prev_small",
	ModelEntityAngle2LoDeltaPrevMedium: "entity_angle2_lo_delta_prev_medium",
	ModelEntityAngle2HiDeltaPrevMedium: "entity_angle2_hi_delta_prev_medium",
	ModelEntityAngle2LoDeltaPrevLarge:  "entity_angle2_lo_delta_prev_large",
	ModelEntityAngle2HiDeltaPrevLarge:  "entity_angle2_hi_delta_prev_large",
	ModelEntityAngle3LoDeltaPrevSmall:  "entity_angle3_lo_delta_prev_small",
	ModelEntityAngle3HiDeltaPrevSmall:  "entity_angle3_hi_delta_prev_small",
	ModelEntityAngle3LoDeltaPrevMedium: "entity_angle3_lo_delta_prev_medium",
	ModelEntityAngle3HiDeltaPrevMedium: "entity_angle3_hi_delta_prev_medium",
	ModelEntityAngle3LoDeltaPrevLarge:  "entity_angle3_lo_delta_prev_large",
	ModelEntityAngle3HiDeltaPrevLarge:  "entity_angle3_hi_delta_prev_large",

	// Player flags, frames, and coordinate high bytes.
	ModelPlayerBitsLoXORPrevNonzero:      "player_bits_lo_xor_prev_nonzero",
	ModelPlayerBitsHiXORPrevNonzero:      "player_bits_hi_xor_prev_nonzero",
	ModelPlayerFrameDeltaPrevOne:         "player_frame_delta_prev_one",
	ModelPlayerFrameDeltaPrevOther:       "player_frame_delta_prev_other",
	ModelPlayerOriginXLoDeltaHighNonzero: "player_origin_x_lo_delta_high_nonzero",
	ModelPlayerOriginYLoDeltaHighNonzero: "player_origin_y_lo_delta_high_nonzero",
	ModelPlayerOriginZLoDeltaHighNonzero: "player_origin_z_lo_delta_high_nonzero",
	ModelPlayerPitchLoDeltaHighNonzero:   "player_pitch_lo_delta_high_nonzero",
	ModelPlayerYawLoDeltaHighNonzero:     "player_yaw_lo_delta_high_nonzero",
	ModelPlayerRollLoDeltaHighNonzero:    "player_roll_lo_delta_high_nonzero",

	// Entity coordinate high bytes.
	ModelEntityOriginXLoDeltaHighNonzero: "entity_origin_x_lo_delta_high_nonzero",
	ModelEntityOriginYLoDeltaHighNonzero: "entity_origin_y_lo_delta_high_nonzero",
	ModelEntityOriginZLoDeltaHighNonzero: "entity_origin_z_lo_delta_high_nonzero",
	ModelEntityAngle1LoDeltaHighNonzero:  "entity_angle1_lo_delta_high_nonzero",
	ModelEntityAngle2LoDeltaHighNonzero:  "entity_angle2_lo_delta_high_nonzero",
	ModelEntityAngle3LoDeltaHighNonzero:  "entity_angle3_lo_delta_high_nonzero",

	// Player angles: negative previous prediction residuals.
	ModelPlayerPitchLoDeltaPrevSmallNegative: "player_pitch_lo_delta_" +
		"prev_small_negative",
	ModelPlayerPitchHiDeltaPrevSmallNegative: "player_pitch_hi_delta_" +
		"prev_small_negative",
	ModelPlayerPitchLoDeltaPrevMediumNegative: "player_pitch_lo_delta_" +
		"prev_medium_negative",
	ModelPlayerPitchHiDeltaPrevMediumNegative: "player_pitch_hi_delta_" +
		"prev_medium_negative",
	ModelPlayerPitchLoDeltaPrevLargeNegative: "player_pitch_lo_delta_" +
		"prev_large_negative",
	ModelPlayerPitchHiDeltaPrevLargeNegative: "player_pitch_hi_delta_" +
		"prev_large_negative",
	ModelPlayerYawLoDeltaPrevSmallNegative: "player_yaw_lo_delta_prev_small_negative",
	ModelPlayerYawHiDeltaPrevSmallNegative: "player_yaw_hi_delta_prev_small_negative",
	ModelPlayerYawLoDeltaPrevMediumNegative: "player_yaw_lo_delta_" +
		"prev_medium_negative",
	ModelPlayerYawHiDeltaPrevMediumNegative: "player_yaw_hi_delta_" +
		"prev_medium_negative",
	ModelPlayerYawLoDeltaPrevLargeNegative: "player_yaw_lo_delta_prev_large_negative",
	ModelPlayerYawHiDeltaPrevLargeNegative: "player_yaw_hi_delta_prev_large_negative",

	// Player origins: negative previous prediction residuals.
	ModelPlayerOriginXLoDeltaPrevSmallNegative: "player_origin_x_lo_delta_" +
		"prev_small_negative",
	ModelPlayerOriginXHiDeltaPrevSmallNegative: "player_origin_x_hi_delta_" +
		"prev_small_negative",
	ModelPlayerOriginXLoDeltaPrevMediumNegative: "player_origin_x_lo_delta_" +
		"prev_medium_negative",
	ModelPlayerOriginXHiDeltaPrevMediumNegative: "player_origin_x_hi_delta_" +
		"prev_medium_negative",
	ModelPlayerOriginXLoDeltaPrevLargeNegative: "player_origin_x_lo_delta_" +
		"prev_large_negative",
	ModelPlayerOriginXHiDeltaPrevLargeNegative: "player_origin_x_hi_delta_" +
		"prev_large_negative",
	ModelPlayerOriginYLoDeltaPrevSmallNegative: "player_origin_y_lo_delta_" +
		"prev_small_negative",
	ModelPlayerOriginYHiDeltaPrevSmallNegative: "player_origin_y_hi_delta_" +
		"prev_small_negative",
	ModelPlayerOriginYLoDeltaPrevMediumNegative: "player_origin_y_lo_delta_" +
		"prev_medium_negative",
	ModelPlayerOriginYHiDeltaPrevMediumNegative: "player_origin_y_hi_delta_" +
		"prev_medium_negative",
	ModelPlayerOriginYLoDeltaPrevLargeNegative: "player_origin_y_lo_delta_" +
		"prev_large_negative",
	ModelPlayerOriginYHiDeltaPrevLargeNegative: "player_origin_y_hi_delta_" +
		"prev_large_negative",
	ModelPlayerOriginZLoDeltaPrevSmallNegative: "player_origin_z_lo_delta_" +
		"prev_small_negative",
	ModelPlayerOriginZHiDeltaPrevSmallNegative: "player_origin_z_hi_delta_" +
		"prev_small_negative",
	ModelPlayerOriginZLoDeltaPrevMediumNegative: "player_origin_z_lo_delta_" +
		"prev_medium_negative",
	ModelPlayerOriginZHiDeltaPrevMediumNegative: "player_origin_z_hi_delta_" +
		"prev_medium_negative",
	ModelPlayerOriginZLoDeltaPrevLargeNegative: "player_origin_z_lo_delta_" +
		"prev_large_negative",
	ModelPlayerOriginZHiDeltaPrevLargeNegative: "player_origin_z_hi_delta_" +
		"prev_large_negative",

	// Entity origins: negative previous prediction residuals.
	ModelEntityOriginXLoDeltaPrevSmallNegative: "entity_origin_x_lo_delta_" +
		"prev_small_negative",
	ModelEntityOriginXHiDeltaPrevSmallNegative: "entity_origin_x_hi_delta_" +
		"prev_small_negative",
	ModelEntityOriginXLoDeltaPrevMediumNegative: "entity_origin_x_lo_delta_" +
		"prev_medium_negative",
	ModelEntityOriginXHiDeltaPrevMediumNegative: "entity_origin_x_hi_delta_" +
		"prev_medium_negative",
	ModelEntityOriginXLoDeltaPrevLargeNegative: "entity_origin_x_lo_delta_" +
		"prev_large_negative",
	ModelEntityOriginXHiDeltaPrevLargeNegative: "entity_origin_x_hi_delta_" +
		"prev_large_negative",
	ModelEntityOriginYLoDeltaPrevSmallNegative: "entity_origin_y_lo_delta_" +
		"prev_small_negative",
	ModelEntityOriginYHiDeltaPrevSmallNegative: "entity_origin_y_hi_delta_" +
		"prev_small_negative",
	ModelEntityOriginYLoDeltaPrevMediumNegative: "entity_origin_y_lo_delta_" +
		"prev_medium_negative",
	ModelEntityOriginYHiDeltaPrevMediumNegative: "entity_origin_y_hi_delta_" +
		"prev_medium_negative",
	ModelEntityOriginYLoDeltaPrevLargeNegative: "entity_origin_y_lo_delta_" +
		"prev_large_negative",
	ModelEntityOriginYHiDeltaPrevLargeNegative: "entity_origin_y_hi_delta_" +
		"prev_large_negative",
	ModelEntityOriginZLoDeltaPrevSmallNegative: "entity_origin_z_lo_delta_" +
		"prev_small_negative",
	ModelEntityOriginZHiDeltaPrevSmallNegative: "entity_origin_z_hi_delta_" +
		"prev_small_negative",
	ModelEntityOriginZLoDeltaPrevMediumNegative: "entity_origin_z_lo_delta_" +
		"prev_medium_negative",
	ModelEntityOriginZHiDeltaPrevMediumNegative: "entity_origin_z_hi_delta_" +
		"prev_medium_negative",
	ModelEntityOriginZLoDeltaPrevLargeNegative: "entity_origin_z_lo_delta_" +
		"prev_large_negative",
	ModelEntityOriginZHiDeltaPrevLargeNegative: "entity_origin_z_hi_delta_" +
		"prev_large_negative",

	// Entity angles: negative previous prediction residuals.
	ModelEntityAngle1LoDeltaPrevSmallNegative: "entity_angle1_lo_delta_" +
		"prev_small_negative",
	ModelEntityAngle1HiDeltaPrevSmallNegative: "entity_angle1_hi_delta_" +
		"prev_small_negative",
	ModelEntityAngle1LoDeltaPrevMediumNegative: "entity_angle1_lo_delta_" +
		"prev_medium_negative",
	ModelEntityAngle1HiDeltaPrevMediumNegative: "entity_angle1_hi_delta_" +
		"prev_medium_negative",
	ModelEntityAngle1LoDeltaPrevLargeNegative: "entity_angle1_lo_delta_" +
		"prev_large_negative",
	ModelEntityAngle1HiDeltaPrevLargeNegative: "entity_angle1_hi_delta_" +
		"prev_large_negative",
	ModelEntityAngle2LoDeltaPrevSmallNegative: "entity_angle2_lo_delta_" +
		"prev_small_negative",
	ModelEntityAngle2HiDeltaPrevSmallNegative: "entity_angle2_hi_delta_" +
		"prev_small_negative",
	ModelEntityAngle2LoDeltaPrevMediumNegative: "entity_angle2_lo_delta_" +
		"prev_medium_negative",
	ModelEntityAngle2HiDeltaPrevMediumNegative: "entity_angle2_hi_delta_" +
		"prev_medium_negative",
	ModelEntityAngle2LoDeltaPrevLargeNegative: "entity_angle2_lo_delta_" +
		"prev_large_negative",
	ModelEntityAngle2HiDeltaPrevLargeNegative: "entity_angle2_hi_delta_" +
		"prev_large_negative",
	ModelEntityAngle3LoDeltaPrevSmallNegative: "entity_angle3_lo_delta_" +
		"prev_small_negative",
	ModelEntityAngle3HiDeltaPrevSmallNegative: "entity_angle3_hi_delta_" +
		"prev_small_negative",
	ModelEntityAngle3LoDeltaPrevMediumNegative: "entity_angle3_lo_delta_" +
		"prev_medium_negative",
	ModelEntityAngle3HiDeltaPrevMediumNegative: "entity_angle3_hi_delta_" +
		"prev_medium_negative",
	ModelEntityAngle3LoDeltaPrevLargeNegative: "entity_angle3_lo_delta_" +
		"prev_large_negative",
	ModelEntityAngle3HiDeltaPrevLargeNegative: "entity_angle3_hi_delta_" +
		"prev_large_negative",

	// Run counts, timestamp history, and extended entity flags.
	ModelPlayerRunLengthDelta:    "player_run_length_delta",
	ModelEntityCountDelta:        "entity_count_delta",
	ModelTimestampKind4PrevKind0: "timestamp_kind_4_prev_kind_0",
	ModelTimestampKind4PrevKind1: "timestamp_kind_4_prev_kind_1",
	ModelTimestampKind4PrevKind2: "timestamp_kind_4_prev_kind_2",
	ModelTimestampKind4PrevKind3: "timestamp_kind_4_prev_kind_3",
	ModelTimestampKind4PrevKind4: "timestamp_kind_4_prev_kind_4",
	ModelTimestampKind4PrevKind5: "timestamp_kind_4_prev_kind_5",
	ModelTimestampKind4PrevKind6: "timestamp_kind_4_prev_kind_6",
	ModelTimestampKind6PrevKind0: "timestamp_kind_6_prev_kind_0",
	ModelTimestampKind6PrevKind1: "timestamp_kind_6_prev_kind_1",
	ModelTimestampKind6PrevKind2: "timestamp_kind_6_prev_kind_2",
	ModelTimestampKind6PrevKind3: "timestamp_kind_6_prev_kind_3",
	ModelTimestampKind6PrevKind4: "timestamp_kind_6_prev_kind_4",
	ModelTimestampKind6PrevKind5: "timestamp_kind_6_prev_kind_5",
	ModelTimestampKind6PrevKind6: "timestamp_kind_6_prev_kind_6",
	ModelEntityEvenMorePresent:   "entity_even_more_present",

	// Sound operations.
	ModelOperationKindAfterSound: "operation_kind_after_sound",
	ModelSoundChannelLoDelta:     "sound_channel_lo_delta",
	ModelSoundChannelHiDelta:     "sound_channel_hi_delta",
	ModelSoundVolume:             "sound_volume",
	ModelSoundAttenuation:        "sound_attenuation",
	ModelSoundNumberDelta:        "sound_number_delta",
	ModelSoundCoordSize:          "sound_coord_size",
	ModelSoundCoordXByte0:        "sound_coord_x_byte_0",
	ModelSoundCoordXByte1:        "sound_coord_x_byte_1",
	ModelSoundCoordXByte2:        "sound_coord_x_byte_2",
	ModelSoundCoordXByte3:        "sound_coord_x_byte_3",
	ModelSoundCoordYByte0:        "sound_coord_y_byte_0",
	ModelSoundCoordYByte1:        "sound_coord_y_byte_1",
	ModelSoundCoordYByte2:        "sound_coord_y_byte_2",
	ModelSoundCoordYByte3:        "sound_coord_y_byte_3",
	ModelSoundCoordZByte0:        "sound_coord_z_byte_0",
	ModelSoundCoordZByte1:        "sound_coord_z_byte_1",
	ModelSoundCoordZByte2:        "sound_coord_z_byte_2",
	ModelSoundCoordZByte3:        "sound_coord_z_byte_3",

	// Temporary entity operations.
	ModelOperationKindAfterTempEntity: "operation_kind_after_tempentity",
	ModelTempEntityType:               "tempentity_type",
	ModelTempEntityCount:              "tempentity_count",
	ModelTempEntityLoDelta:            "tempentity_entity_lo_delta",
	ModelTempEntityHiDelta:            "tempentity_entity_hi_delta",
	ModelTempEntityCoordSize:          "tempentity_coord_size",
	ModelTempEntityCoordXByte0:        "tempentity_coord_x_byte_0",
	ModelTempEntityCoordXByte1:        "tempentity_coord_x_byte_1",
	ModelTempEntityCoordXByte2:        "tempentity_coord_x_byte_2",
	ModelTempEntityCoordXByte3:        "tempentity_coord_x_byte_3",
	ModelTempEntityCoordYByte0:        "tempentity_coord_y_byte_0",
	ModelTempEntityCoordYByte1:        "tempentity_coord_y_byte_1",
	ModelTempEntityCoordYByte2:        "tempentity_coord_y_byte_2",
	ModelTempEntityCoordYByte3:        "tempentity_coord_y_byte_3",
	ModelTempEntityCoordZByte0:        "tempentity_coord_z_byte_0",
	ModelTempEntityCoordZByte1:        "tempentity_coord_z_byte_1",
	ModelTempEntityCoordZByte2:        "tempentity_coord_z_byte_2",
	ModelTempEntityCoordZByte3:        "tempentity_coord_z_byte_3",

	// Service operations with fixed sizes.
	ModelOperationKindAfterFixed:  "operation_kind_after_fixed",
	ModelUpdateStatIndexDelta:     "updatestat_index_delta",
	ModelUpdateStatValueDelta:     "updatestat_value_delta",
	ModelUpdateStatLongIndexDelta: "updatestatlong_index_delta",
	ModelUpdateStatLongDeltaByte0: "updatestatlong_delta_byte_0",
	ModelUpdateStatLongDeltaByte1: "updatestatlong_delta_byte_1",
	ModelUpdateStatLongDeltaByte2: "updatestatlong_delta_byte_2",
	ModelUpdateStatLongDeltaByte3: "updatestatlong_delta_byte_3",
	ModelFixedPlayerIndexDelta:    "fixed_player_index_delta",
	ModelUpdateFragsLoDelta:       "updatefrags_lo_delta",
	ModelUpdateFragsHiDelta:       "updatefrags_hi_delta",
	ModelUpdatePingLoDelta:        "updateping_lo_delta",
	ModelUpdatePingHiDelta:        "updateping_hi_delta",
	ModelUpdatePLDelta:            "updatepl_delta",
	ModelMuzzleFlashLoDelta:       "muzzleflash_lo_delta",
	ModelMuzzleFlashHiDelta:       "muzzleflash_hi_delta",

	// Damage operations.
	ModelOperationKindAfterDamage: "operation_kind_after_damage",
	ModelDamageArmor:              "damage_armor",
	ModelDamageBlood:              "damage_blood",
	ModelDamageCoordSize:          "damage_coord_size",
	ModelDamageHasPosition:        "damage_has_position",
	ModelDamageCoordXByte0:        "damage_coord_x_byte_0",
	ModelDamageCoordXByte1:        "damage_coord_x_byte_1",
	ModelDamageCoordXByte2:        "damage_coord_x_byte_2",
	ModelDamageCoordXByte3:        "damage_coord_x_byte_3",
	ModelDamageCoordYByte0:        "damage_coord_y_byte_0",
	ModelDamageCoordYByte1:        "damage_coord_y_byte_1",
	ModelDamageCoordYByte2:        "damage_coord_y_byte_2",
	ModelDamageCoordYByte3:        "damage_coord_y_byte_3",
	ModelDamageCoordZByte0:        "damage_coord_z_byte_0",
	ModelDamageCoordZByte1:        "damage_coord_z_byte_1",
	ModelDamageCoordZByte2:        "damage_coord_z_byte_2",
	ModelDamageCoordZByte3:        "damage_coord_z_byte_3",

	// More specific previous operation contexts.
	ModelOperationKindAfterCenterPrint:    "operation_kind_after_centerprint",
	ModelOperationKindAfterStuffText:      "operation_kind_after_stufftext",
	ModelOperationKindAfterPrint:          "operation_kind_after_print",
	ModelOperationKindAfterUpdateStat:     "operation_kind_after_updatestat",
	ModelOperationKindAfterUpdateFrags:    "operation_kind_after_updatefrags",
	ModelOperationKindAfterUpdatePing:     "operation_kind_after_updateping",
	ModelOperationKindAfterUpdateStatLong: "operation_kind_after_updatestatlong",
	ModelOperationKindAfterMuzzleFlash:    "operation_kind_after_muzzleflash",
	ModelOperationKindAfterUpdatePL:       "operation_kind_after_updatepl",

	// Coordinate base selection.
	ModelSoundCoordBase:          "sound_coord_base",
	ModelTempEntityBeamCoordBase: "tempentity_beam_coord_base",
	ModelDamageCoordBase:         "damage_coord_base",

	// Sound coordinates relative to an entity.
	ModelSoundEntityCoordXByte0: "sound_entity_coord_x_byte_0",
	ModelSoundEntityCoordXByte1: "sound_entity_coord_x_byte_1",
	ModelSoundEntityCoordXByte2: "sound_entity_coord_x_byte_2",
	ModelSoundEntityCoordXByte3: "sound_entity_coord_x_byte_3",
	ModelSoundEntityCoordYByte0: "sound_entity_coord_y_byte_0",
	ModelSoundEntityCoordYByte1: "sound_entity_coord_y_byte_1",
	ModelSoundEntityCoordYByte2: "sound_entity_coord_y_byte_2",
	ModelSoundEntityCoordYByte3: "sound_entity_coord_y_byte_3",
	ModelSoundEntityCoordZByte0: "sound_entity_coord_z_byte_0",
	ModelSoundEntityCoordZByte1: "sound_entity_coord_z_byte_1",
	ModelSoundEntityCoordZByte2: "sound_entity_coord_z_byte_2",
	ModelSoundEntityCoordZByte3: "sound_entity_coord_z_byte_3",

	// Coordinate low bytes conditioned on nonzero high bytes.
	ModelSoundCoordXLoHighNonzero:       "sound_coord_x_lo_high_nonzero",
	ModelSoundCoordYLoHighNonzero:       "sound_coord_y_lo_high_nonzero",
	ModelSoundCoordZLoHighNonzero:       "sound_coord_z_lo_high_nonzero",
	ModelSoundEntityCoordXLoHighNonzero: "sound_entity_coord_x_lo_high_nonzero",
	ModelSoundEntityCoordYLoHighNonzero: "sound_entity_coord_y_lo_high_nonzero",
	ModelSoundEntityCoordZLoHighNonzero: "sound_entity_coord_z_lo_high_nonzero",
	ModelTempEntityCoordXLoHighNonzero:  "tempentity_coord_x_lo_high_nonzero",
	ModelTempEntityCoordYLoHighNonzero:  "tempentity_coord_y_lo_high_nonzero",
	ModelTempEntityCoordZLoHighNonzero:  "tempentity_coord_z_lo_high_nonzero",
	ModelDamageCoordXLoHighNonzero:      "damage_coord_x_lo_high_nonzero",
	ModelDamageCoordYLoHighNonzero:      "damage_coord_y_lo_high_nonzero",
	ModelDamageCoordZLoHighNonzero:      "damage_coord_z_lo_high_nonzero",

	// Player/world sound and beam contexts.
	ModelSoundNumberDeltaPlayer:        "sound_number_delta_player",
	ModelSoundNumberDeltaWorld:         "sound_number_delta_world",
	ModelSoundCoordBasePlayer:          "sound_coord_base_player",
	ModelSoundCoordBaseWorld:           "sound_coord_base_world",
	ModelTempEntityBeamCoordBasePlayer: "tempentity_beam_coord_base_player",
	ModelTempEntityBeamCoordBaseWorld:  "tempentity_beam_coord_base_world",
}
