package codec

import "github.com/osm/quake/demo/mvz/frequency"

var soundCoordinateModels = [3][4]frequency.Model{
	{
		frequency.ModelSoundCoordXByte0,
		frequency.ModelSoundCoordXByte1,
		frequency.ModelSoundCoordXByte2,
		frequency.ModelSoundCoordXByte3,
	},
	{
		frequency.ModelSoundCoordYByte0,
		frequency.ModelSoundCoordYByte1,
		frequency.ModelSoundCoordYByte2,
		frequency.ModelSoundCoordYByte3,
	},
	{
		frequency.ModelSoundCoordZByte0,
		frequency.ModelSoundCoordZByte1,
		frequency.ModelSoundCoordZByte2,
		frequency.ModelSoundCoordZByte3,
	},
}

var soundEntityCoordinateModels = [3][4]frequency.Model{
	{
		frequency.ModelSoundEntityCoordXByte0,
		frequency.ModelSoundEntityCoordXByte1,
		frequency.ModelSoundEntityCoordXByte2,
		frequency.ModelSoundEntityCoordXByte3,
	},
	{
		frequency.ModelSoundEntityCoordYByte0,
		frequency.ModelSoundEntityCoordYByte1,
		frequency.ModelSoundEntityCoordYByte2,
		frequency.ModelSoundEntityCoordYByte3,
	},
	{
		frequency.ModelSoundEntityCoordZByte0,
		frequency.ModelSoundEntityCoordZByte1,
		frequency.ModelSoundEntityCoordZByte2,
		frequency.ModelSoundEntityCoordZByte3,
	},
}

var soundCoordinateLowHighNonzeroModels = [3]frequency.Model{
	frequency.ModelSoundCoordXLoHighNonzero,
	frequency.ModelSoundCoordYLoHighNonzero,
	frequency.ModelSoundCoordZLoHighNonzero,
}

var soundEntityCoordinateLowHighNonzeroModels = [3]frequency.Model{
	frequency.ModelSoundEntityCoordXLoHighNonzero,
	frequency.ModelSoundEntityCoordYLoHighNonzero,
	frequency.ModelSoundEntityCoordZLoHighNonzero,
}

var soundNumberDeltaModels = [3]frequency.Model{
	effectClassZero:   frequency.ModelSoundNumberDelta,
	effectClassPlayer: frequency.ModelSoundNumberDeltaPlayer,
	effectClassOther:  frequency.ModelSoundNumberDeltaWorld,
}

var soundCoordBaseModels = [3]frequency.Model{
	effectClassZero:   frequency.ModelSoundCoordBase,
	effectClassPlayer: frequency.ModelSoundCoordBasePlayer,
	effectClassOther:  frequency.ModelSoundCoordBaseWorld,
}

var soundChannelModels = modelPair{
	frequency.ModelSoundChannelLoDelta,
	frequency.ModelSoundChannelHiDelta,
}
