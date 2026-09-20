package codec

import "github.com/osm/quake/demo/mvz/frequency"

// Context table order is part of the v1 grammar.
var playerIndexContextModels = [4]frequency.Model{
	frequency.ModelPlayerIndex,
	frequency.ModelPlayerIndexPrevSmall,
	frequency.ModelPlayerIndexPrevMedium,
	frequency.ModelPlayerIndexPrevLarge,
}

func playerIndexContext(value byte) int {
	switch {
	case value == 0:
		return 0
	case value <= 2:
		return 1
	case value <= 8:
		return 2
	default:
		return 3
	}
}

func playerIndexModel(state *rangeCodecState) frequency.Model {
	return playerIndexContextModels[playerIndexContext(state.previousPlayerIndexDelta)]
}

func playerBitsModels(state *rangePlayerState) modelPair {
	if state.bitsResidual != 0 {
		return modelPair{
			frequency.ModelPlayerBitsLoXORPrevNonzero,
			frequency.ModelPlayerBitsHiXORPrevNonzero,
		}
	}
	return modelPair{
		frequency.ModelPlayerBitsLoXOR,
		frequency.ModelPlayerBitsHiXOR,
	}
}

func playerFrameModel(state *rangePlayerState) frequency.Model {
	if state.frameResidual == 0 {
		return frequency.ModelPlayerFrameDelta
	}
	if state.frameResidual == 1 {
		return frequency.ModelPlayerFrameDeltaPrevOne
	}
	return frequency.ModelPlayerFrameDeltaPrevOther
}

var playerOriginModels = [3]modelPair{
	{frequency.ModelPlayerOriginXLoDelta, frequency.ModelPlayerOriginXHiDelta},
	{frequency.ModelPlayerOriginYLoDelta, frequency.ModelPlayerOriginYHiDelta},
	{frequency.ModelPlayerOriginZLoDelta, frequency.ModelPlayerOriginZHiDelta},
}

var playerOriginLowHighNonzeroModels = [3]frequency.Model{
	frequency.ModelPlayerOriginXLoDeltaHighNonzero,
	frequency.ModelPlayerOriginYLoDeltaHighNonzero,
	frequency.ModelPlayerOriginZLoDeltaHighNonzero,
}

var playerOriginContextModels = [3][4]modelPair{
	{
		{frequency.ModelPlayerOriginXLoDelta, frequency.ModelPlayerOriginXHiDelta},
		{
			frequency.ModelPlayerOriginXLoDeltaPrevSmall,
			frequency.ModelPlayerOriginXHiDeltaPrevSmall,
		},
		{
			frequency.ModelPlayerOriginXLoDeltaPrevMedium,
			frequency.ModelPlayerOriginXHiDeltaPrevMedium,
		},
		{
			frequency.ModelPlayerOriginXLoDeltaPrevLarge,
			frequency.ModelPlayerOriginXHiDeltaPrevLarge,
		},
	},
	{
		{frequency.ModelPlayerOriginYLoDelta, frequency.ModelPlayerOriginYHiDelta},
		{
			frequency.ModelPlayerOriginYLoDeltaPrevSmall,
			frequency.ModelPlayerOriginYHiDeltaPrevSmall,
		},
		{
			frequency.ModelPlayerOriginYLoDeltaPrevMedium,
			frequency.ModelPlayerOriginYHiDeltaPrevMedium,
		},
		{
			frequency.ModelPlayerOriginYLoDeltaPrevLarge,
			frequency.ModelPlayerOriginYHiDeltaPrevLarge,
		},
	},
	{
		{frequency.ModelPlayerOriginZLoDelta, frequency.ModelPlayerOriginZHiDelta},
		{
			frequency.ModelPlayerOriginZLoDeltaPrevSmall,
			frequency.ModelPlayerOriginZHiDeltaPrevSmall,
		},
		{
			frequency.ModelPlayerOriginZLoDeltaPrevMedium,
			frequency.ModelPlayerOriginZHiDeltaPrevMedium,
		},
		{
			frequency.ModelPlayerOriginZLoDeltaPrevLarge,
			frequency.ModelPlayerOriginZHiDeltaPrevLarge,
		},
	},
}

var playerOriginNegativeContextModels = [3][3]modelPair{
	{
		{
			frequency.ModelPlayerOriginXLoDeltaPrevSmallNegative,
			frequency.ModelPlayerOriginXHiDeltaPrevSmallNegative,
		},
		{
			frequency.ModelPlayerOriginXLoDeltaPrevMediumNegative,
			frequency.ModelPlayerOriginXHiDeltaPrevMediumNegative,
		},
		{
			frequency.ModelPlayerOriginXLoDeltaPrevLargeNegative,
			frequency.ModelPlayerOriginXHiDeltaPrevLargeNegative,
		},
	},
	{
		{
			frequency.ModelPlayerOriginYLoDeltaPrevSmallNegative,
			frequency.ModelPlayerOriginYHiDeltaPrevSmallNegative,
		},
		{
			frequency.ModelPlayerOriginYLoDeltaPrevMediumNegative,
			frequency.ModelPlayerOriginYHiDeltaPrevMediumNegative,
		},
		{
			frequency.ModelPlayerOriginYLoDeltaPrevLargeNegative,
			frequency.ModelPlayerOriginYHiDeltaPrevLargeNegative,
		},
	},
	{
		{
			frequency.ModelPlayerOriginZLoDeltaPrevSmallNegative,
			frequency.ModelPlayerOriginZHiDeltaPrevSmallNegative,
		},
		{
			frequency.ModelPlayerOriginZLoDeltaPrevMediumNegative,
			frequency.ModelPlayerOriginZHiDeltaPrevMediumNegative,
		},
		{
			frequency.ModelPlayerOriginZLoDeltaPrevLargeNegative,
			frequency.ModelPlayerOriginZHiDeltaPrevLargeNegative,
		},
	},
}

var playerAngleModels = [3]modelPair{
	{frequency.ModelPlayerPitchLoDelta, frequency.ModelPlayerPitchHiDelta},
	{frequency.ModelPlayerYawLoDelta, frequency.ModelPlayerYawHiDelta},
	{frequency.ModelPlayerRollLoDelta, frequency.ModelPlayerRollHiDelta},
}

var playerAngleLowHighNonzeroModels = [3]frequency.Model{
	frequency.ModelPlayerPitchLoDeltaHighNonzero,
	frequency.ModelPlayerYawLoDeltaHighNonzero,
	frequency.ModelPlayerRollLoDeltaHighNonzero,
}

var playerAngleContextModels = [2][4]modelPair{
	{
		{frequency.ModelPlayerPitchLoDelta, frequency.ModelPlayerPitchHiDelta},
		{
			frequency.ModelPlayerPitchLoDeltaPrevSmall,
			frequency.ModelPlayerPitchHiDeltaPrevSmall,
		},
		{
			frequency.ModelPlayerPitchLoDeltaPrevMedium,
			frequency.ModelPlayerPitchHiDeltaPrevMedium,
		},
		{
			frequency.ModelPlayerPitchLoDeltaPrevLarge,
			frequency.ModelPlayerPitchHiDeltaPrevLarge,
		},
	},
	{
		{frequency.ModelPlayerYawLoDelta, frequency.ModelPlayerYawHiDelta},
		{frequency.ModelPlayerYawLoDeltaPrevSmall, frequency.ModelPlayerYawHiDeltaPrevSmall},
		{
			frequency.ModelPlayerYawLoDeltaPrevMedium,
			frequency.ModelPlayerYawHiDeltaPrevMedium,
		},
		{frequency.ModelPlayerYawLoDeltaPrevLarge, frequency.ModelPlayerYawHiDeltaPrevLarge},
	},
}

var playerAngleNegativeContextModels = [2][3]modelPair{
	{
		{
			frequency.ModelPlayerPitchLoDeltaPrevSmallNegative,
			frequency.ModelPlayerPitchHiDeltaPrevSmallNegative,
		},
		{
			frequency.ModelPlayerPitchLoDeltaPrevMediumNegative,
			frequency.ModelPlayerPitchHiDeltaPrevMediumNegative,
		},
		{
			frequency.ModelPlayerPitchLoDeltaPrevLargeNegative,
			frequency.ModelPlayerPitchHiDeltaPrevLargeNegative,
		},
	},
	{
		{
			frequency.ModelPlayerYawLoDeltaPrevSmallNegative,
			frequency.ModelPlayerYawHiDeltaPrevSmallNegative,
		},
		{
			frequency.ModelPlayerYawLoDeltaPrevMediumNegative,
			frequency.ModelPlayerYawHiDeltaPrevMediumNegative,
		},
		{
			frequency.ModelPlayerYawLoDeltaPrevLargeNegative,
			frequency.ModelPlayerYawHiDeltaPrevLargeNegative,
		},
	},
}

func playerAngleModelsFor(
	state *rangePlayerState,
	axis int,
) modelPair {
	if axis >= len(playerAngleContextModels) {
		return playerAngleModels[axis]
	}
	residual := state.angleResidual[axis]
	context := residualMagnitudeContext16(residual)
	if context != 0 && int16(residual) < 0 {
		return playerAngleNegativeContextModels[axis][context-1]
	}
	return playerAngleContextModels[axis][context]
}

func playerOriginModelsFor(
	state *rangePlayerState,
	axis int,
	coordSize int,
) modelPair {
	if coordSize != 2 {
		return playerOriginModels[axis]
	}
	residual := uint16(state.originResidual[axis])
	context := residualMagnitudeContext16(residual)
	if context != 0 && int16(residual) < 0 {
		return playerOriginNegativeContextModels[axis][context-1]
	}
	return playerOriginContextModels[axis][context]
}
