package codec

import "github.com/osm/quake/demo/mvz/frequency"

// Magnitude contexts are zero, small, medium, and large. Negative tables omit
// the zero case, so their indices are one lower. Table order is part of v1.
var entityOriginModels = [3]modelPair{
	{frequency.ModelEntityOriginXLoDelta, frequency.ModelEntityOriginXHiDelta},
	{frequency.ModelEntityOriginYLoDelta, frequency.ModelEntityOriginYHiDelta},
	{frequency.ModelEntityOriginZLoDelta, frequency.ModelEntityOriginZHiDelta},
}

var entityOriginLowHighNonzeroModels = [3]frequency.Model{
	frequency.ModelEntityOriginXLoDeltaHighNonzero,
	frequency.ModelEntityOriginYLoDeltaHighNonzero,
	frequency.ModelEntityOriginZLoDeltaHighNonzero,
}

var entityOriginContextModels = [3][4]modelPair{
	{
		{frequency.ModelEntityOriginXLoDelta, frequency.ModelEntityOriginXHiDelta},
		{
			frequency.ModelEntityOriginXLoDeltaPrevSmall,
			frequency.ModelEntityOriginXHiDeltaPrevSmall,
		},
		{
			frequency.ModelEntityOriginXLoDeltaPrevMedium,
			frequency.ModelEntityOriginXHiDeltaPrevMedium,
		},
		{
			frequency.ModelEntityOriginXLoDeltaPrevLarge,
			frequency.ModelEntityOriginXHiDeltaPrevLarge,
		},
	},
	{
		{frequency.ModelEntityOriginYLoDelta, frequency.ModelEntityOriginYHiDelta},
		{
			frequency.ModelEntityOriginYLoDeltaPrevSmall,
			frequency.ModelEntityOriginYHiDeltaPrevSmall,
		},
		{
			frequency.ModelEntityOriginYLoDeltaPrevMedium,
			frequency.ModelEntityOriginYHiDeltaPrevMedium,
		},
		{
			frequency.ModelEntityOriginYLoDeltaPrevLarge,
			frequency.ModelEntityOriginYHiDeltaPrevLarge,
		},
	},
	{
		{frequency.ModelEntityOriginZLoDelta, frequency.ModelEntityOriginZHiDelta},
		{
			frequency.ModelEntityOriginZLoDeltaPrevSmall,
			frequency.ModelEntityOriginZHiDeltaPrevSmall,
		},
		{
			frequency.ModelEntityOriginZLoDeltaPrevMedium,
			frequency.ModelEntityOriginZHiDeltaPrevMedium,
		},
		{
			frequency.ModelEntityOriginZLoDeltaPrevLarge,
			frequency.ModelEntityOriginZHiDeltaPrevLarge,
		},
	},
}

var entityOriginNegativeContextModels = [3][3]modelPair{
	{
		{
			frequency.ModelEntityOriginXLoDeltaPrevSmallNegative,
			frequency.ModelEntityOriginXHiDeltaPrevSmallNegative,
		},
		{
			frequency.ModelEntityOriginXLoDeltaPrevMediumNegative,
			frequency.ModelEntityOriginXHiDeltaPrevMediumNegative,
		},
		{
			frequency.ModelEntityOriginXLoDeltaPrevLargeNegative,
			frequency.ModelEntityOriginXHiDeltaPrevLargeNegative,
		},
	},
	{
		{
			frequency.ModelEntityOriginYLoDeltaPrevSmallNegative,
			frequency.ModelEntityOriginYHiDeltaPrevSmallNegative,
		},
		{
			frequency.ModelEntityOriginYLoDeltaPrevMediumNegative,
			frequency.ModelEntityOriginYHiDeltaPrevMediumNegative,
		},
		{
			frequency.ModelEntityOriginYLoDeltaPrevLargeNegative,
			frequency.ModelEntityOriginYHiDeltaPrevLargeNegative,
		},
	},
	{
		{
			frequency.ModelEntityOriginZLoDeltaPrevSmallNegative,
			frequency.ModelEntityOriginZHiDeltaPrevSmallNegative,
		},
		{
			frequency.ModelEntityOriginZLoDeltaPrevMediumNegative,
			frequency.ModelEntityOriginZHiDeltaPrevMediumNegative,
		},
		{
			frequency.ModelEntityOriginZLoDeltaPrevLargeNegative,
			frequency.ModelEntityOriginZHiDeltaPrevLargeNegative,
		},
	},
}

func entityOriginModelsFor(
	state *rangeEntityState,
	axis int,
	coordSize int,
) modelPair {
	if coordSize != 2 {
		return entityOriginModels[axis]
	}
	residual := uint16(state.originResidual[axis])
	context := residualMagnitudeContext16(residual)
	if context != 0 && int16(residual) < 0 {
		return entityOriginNegativeContextModels[axis][context-1]
	}
	return entityOriginContextModels[axis][context]
}

var entityAngleContextModels = [3][4]modelPair{
	{
		{frequency.ModelEntityAngle1LoDelta, frequency.ModelEntityAngle1HiDelta},
		{
			frequency.ModelEntityAngle1LoDeltaPrevSmall,
			frequency.ModelEntityAngle1HiDeltaPrevSmall,
		},
		{
			frequency.ModelEntityAngle1LoDeltaPrevMedium,
			frequency.ModelEntityAngle1HiDeltaPrevMedium,
		},
		{
			frequency.ModelEntityAngle1LoDeltaPrevLarge,
			frequency.ModelEntityAngle1HiDeltaPrevLarge,
		},
	},
	{
		{frequency.ModelEntityAngle2LoDelta, frequency.ModelEntityAngle2HiDelta},
		{
			frequency.ModelEntityAngle2LoDeltaPrevSmall,
			frequency.ModelEntityAngle2HiDeltaPrevSmall,
		},
		{
			frequency.ModelEntityAngle2LoDeltaPrevMedium,
			frequency.ModelEntityAngle2HiDeltaPrevMedium,
		},
		{
			frequency.ModelEntityAngle2LoDeltaPrevLarge,
			frequency.ModelEntityAngle2HiDeltaPrevLarge,
		},
	},
	{
		{frequency.ModelEntityAngle3LoDelta, frequency.ModelEntityAngle3HiDelta},
		{
			frequency.ModelEntityAngle3LoDeltaPrevSmall,
			frequency.ModelEntityAngle3HiDeltaPrevSmall,
		},
		{
			frequency.ModelEntityAngle3LoDeltaPrevMedium,
			frequency.ModelEntityAngle3HiDeltaPrevMedium,
		},
		{
			frequency.ModelEntityAngle3LoDeltaPrevLarge,
			frequency.ModelEntityAngle3HiDeltaPrevLarge,
		},
	},
}

var entityAngleLowHighNonzeroModels = [3]frequency.Model{
	frequency.ModelEntityAngle1LoDeltaHighNonzero,
	frequency.ModelEntityAngle2LoDeltaHighNonzero,
	frequency.ModelEntityAngle3LoDeltaHighNonzero,
}

var entityAngleNegativeContextModels = [3][3]modelPair{
	{
		{
			frequency.ModelEntityAngle1LoDeltaPrevSmallNegative,
			frequency.ModelEntityAngle1HiDeltaPrevSmallNegative,
		},
		{
			frequency.ModelEntityAngle1LoDeltaPrevMediumNegative,
			frequency.ModelEntityAngle1HiDeltaPrevMediumNegative,
		},
		{
			frequency.ModelEntityAngle1LoDeltaPrevLargeNegative,
			frequency.ModelEntityAngle1HiDeltaPrevLargeNegative,
		},
	},
	{
		{
			frequency.ModelEntityAngle2LoDeltaPrevSmallNegative,
			frequency.ModelEntityAngle2HiDeltaPrevSmallNegative,
		},
		{
			frequency.ModelEntityAngle2LoDeltaPrevMediumNegative,
			frequency.ModelEntityAngle2HiDeltaPrevMediumNegative,
		},
		{
			frequency.ModelEntityAngle2LoDeltaPrevLargeNegative,
			frequency.ModelEntityAngle2HiDeltaPrevLargeNegative,
		},
	},
	{
		{
			frequency.ModelEntityAngle3LoDeltaPrevSmallNegative,
			frequency.ModelEntityAngle3HiDeltaPrevSmallNegative,
		},
		{
			frequency.ModelEntityAngle3LoDeltaPrevMediumNegative,
			frequency.ModelEntityAngle3HiDeltaPrevMediumNegative,
		},
		{
			frequency.ModelEntityAngle3LoDeltaPrevLargeNegative,
			frequency.ModelEntityAngle3HiDeltaPrevLargeNegative,
		},
	},
}

func entityAngleModelsFor(
	state *rangeEntityState,
	axis, angleSize int,
) modelPair {
	residual := state.angleResidual[axis]
	context := residualMagnitudeContext16(residual)
	negative := int16(residual) < 0
	if angleSize == 1 {
		context = residualMagnitudeContext8(byte(residual))
		negative = int8(byte(residual)) < 0
	}
	if context != 0 && negative {
		return entityAngleNegativeContextModels[axis][context-1]
	}
	return entityAngleContextModels[axis][context]
}

var entityNumberContextModels = [4]modelPair{
	{frequency.ModelEntityNumberLoDelta, frequency.ModelEntityNumberHiDelta},
	{
		frequency.ModelEntityNumberLoDeltaPrevSmall,
		frequency.ModelEntityNumberHiDeltaPrevSmall,
	},
	{
		frequency.ModelEntityNumberLoDeltaPrevMedium,
		frequency.ModelEntityNumberHiDeltaPrevMedium,
	},
	{
		frequency.ModelEntityNumberLoDeltaPrevLarge,
		frequency.ModelEntityNumberHiDeltaPrevLarge,
	},
}

var entityFlagModels = [3]frequency.Model{
	frequency.ModelEntityMoreBitsXOR,
	frequency.ModelEntityEvenMoreBitsXOR,
	frequency.ModelEntityYetMoreBitsXOR,
}
