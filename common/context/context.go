package context

import (
	"sync"

	"github.com/osm/quake/protocol"
)

type Context struct {
	angleSizeMu sync.Mutex
	angleSize   uint8

	coordSizeMU sync.Mutex
	coordSize   uint8

	protocolVersioncolVersionMu sync.Mutex
	protocolVersion             uint32

	isMVDEnabledMu sync.Mutex
	isMVDEnabled   bool

	mvdProtocolExtensionMu sync.Mutex
	mvdProtocolExtension   uint32

	isFTEEnabledMu sync.Mutex
	isFTEEnabled   bool

	fteProtocolExtensionMu sync.Mutex
	fteProtocolExtension   uint32

	isFTE2EnabledMu sync.Mutex
	isFTE2Enabled   bool

	fte2ProtocolExtensionMu sync.Mutex
	fte2ProtocolExtension   uint32

	isZQuakeEnabledMu sync.Mutex
	isZQuakeEnabled   bool

	zQuakeProtocolExtensionMu sync.Mutex
	zQuakeProtocolExtension   uint32

	isDemMu sync.Mutex
	isDem   bool

	isQWDMu sync.Mutex
	isQWD   bool

	isMVDMu sync.Mutex
	isMVD   bool
}

func New(opts ...Option) *Context {
	ctx := &Context{
		angleSize: 1,
		coordSize: 2,
	}

	for _, opt := range opts {
		opt(ctx)
	}

	return ctx
}

func (ctx *Context) GetAngleSize() uint8 {
	ctx.angleSizeMu.Lock()
	defer ctx.angleSizeMu.Unlock()
	return ctx.angleSize
}

func (ctx *Context) SetAngleSize(v uint8) {
	ctx.angleSizeMu.Lock()
	defer ctx.angleSizeMu.Unlock()
	ctx.angleSize = v
}

func (ctx *Context) GetCoordSize() uint8 {
	ctx.coordSizeMU.Lock()
	defer ctx.coordSizeMU.Unlock()
	return ctx.coordSize
}

func (ctx *Context) SetCoordSize(v uint8) {
	ctx.coordSizeMU.Lock()
	defer ctx.coordSizeMU.Unlock()
	ctx.coordSize = v
}

func (ctx *Context) GetProtocolVersion() uint32 {
	ctx.protocolVersioncolVersionMu.Lock()
	defer ctx.protocolVersioncolVersionMu.Unlock()
	return ctx.protocolVersion
}

func (ctx *Context) SetProtocolVersion(v uint32) {
	ctx.protocolVersioncolVersionMu.Lock()
	defer ctx.protocolVersioncolVersionMu.Unlock()
	ctx.protocolVersion = v
}

func (ctx *Context) GetIsMVDEnabled() bool {
	ctx.isMVDEnabledMu.Lock()
	defer ctx.isMVDEnabledMu.Unlock()
	return ctx.isMVDEnabled
}

func (ctx *Context) SetIsMVDEnabled(v bool) {
	ctx.isMVDEnabledMu.Lock()
	defer ctx.isMVDEnabledMu.Unlock()
	ctx.isMVDEnabled = v
}

func (ctx *Context) GetMVDProtocolExtension() uint32 {
	ctx.mvdProtocolExtensionMu.Lock()
	defer ctx.mvdProtocolExtensionMu.Unlock()
	return ctx.mvdProtocolExtension
}

func (ctx *Context) SetMVDProtocolExtension(v uint32) {
	ctx.mvdProtocolExtensionMu.Lock()
	defer ctx.mvdProtocolExtensionMu.Unlock()
	ctx.mvdProtocolExtension = v
}

func (ctx *Context) GetIsFTEEnabled() bool {
	ctx.isFTEEnabledMu.Lock()
	defer ctx.isFTEEnabledMu.Unlock()
	return ctx.isFTEEnabled
}

func (ctx *Context) SetIsFTEEnabled(v bool) {
	ctx.isFTEEnabledMu.Lock()
	defer ctx.isFTEEnabledMu.Unlock()
	ctx.isFTEEnabled = v
}

func (ctx *Context) GetFTEProtocolExtension() uint32 {
	ctx.fteProtocolExtensionMu.Lock()
	defer ctx.fteProtocolExtensionMu.Unlock()
	return ctx.fteProtocolExtension
}

func (ctx *Context) SetFTEProtocolExtension(v uint32) {
	ctx.fteProtocolExtensionMu.Lock()
	defer ctx.fteProtocolExtensionMu.Unlock()
	ctx.fteProtocolExtension = v
}

func (ctx *Context) GetIsFTE2Enabled() bool {
	ctx.isFTE2EnabledMu.Lock()
	defer ctx.isFTE2EnabledMu.Unlock()
	return ctx.isFTE2Enabled
}

func (ctx *Context) SetIsFTE2Enabled(v bool) {
	ctx.isFTE2EnabledMu.Lock()
	defer ctx.isFTE2EnabledMu.Unlock()
	ctx.isFTE2Enabled = v
}

func (ctx *Context) GetFTE2ProtocolExtension() uint32 {
	ctx.fte2ProtocolExtensionMu.Lock()
	defer ctx.fte2ProtocolExtensionMu.Unlock()
	return ctx.fte2ProtocolExtension
}

func (ctx *Context) SetFTE2ProtocolExtension(v uint32) {
	ctx.fte2ProtocolExtensionMu.Lock()
	defer ctx.fte2ProtocolExtensionMu.Unlock()
	ctx.fte2ProtocolExtension = v
}

func (ctx *Context) GetIsZQuakeEnabled() bool {
	ctx.isZQuakeEnabledMu.Lock()
	defer ctx.isZQuakeEnabledMu.Unlock()
	return ctx.isZQuakeEnabled
}

func (ctx *Context) SetIsZQuakeEnabled(v bool) {
	ctx.isZQuakeEnabledMu.Lock()
	defer ctx.isZQuakeEnabledMu.Unlock()
	ctx.isZQuakeEnabled = v
}

func (ctx *Context) GetZQuakeProtocolExtension() uint32 {
	ctx.zQuakeProtocolExtensionMu.Lock()
	defer ctx.zQuakeProtocolExtensionMu.Unlock()
	return ctx.zQuakeProtocolExtension
}

func (ctx *Context) SetZQuakeProtocolExtension(v uint32) {
	ctx.zQuakeProtocolExtensionMu.Lock()
	defer ctx.zQuakeProtocolExtensionMu.Unlock()
	ctx.zQuakeProtocolExtension = v
}

func (ctx *Context) GetIsDem() bool {
	ctx.isDemMu.Lock()
	defer ctx.isDemMu.Unlock()
	return ctx.isDem
}

func (ctx *Context) SetIsDem(v bool) {
	ctx.isDemMu.Lock()
	defer ctx.isDemMu.Unlock()
	ctx.isDem = v
}

func (ctx *Context) GetIsQWD() bool {
	ctx.isQWDMu.Lock()
	defer ctx.isQWDMu.Unlock()
	return ctx.isQWD
}

func (ctx *Context) SetIsQWD(v bool) {
	ctx.isQWDMu.Lock()
	defer ctx.isQWDMu.Unlock()
	ctx.isQWD = v
}

func (ctx *Context) GetIsMVD() bool {
	ctx.isMVDMu.Lock()
	defer ctx.isMVDMu.Unlock()
	return ctx.isMVD
}

func (ctx *Context) SetIsMVD(v bool) {
	ctx.isMVDMu.Lock()
	defer ctx.isMVDMu.Unlock()
	ctx.isMVD = v
}

func (ctx *Context) GetIsNQ() bool {
	return ctx.GetIsDem() || ctx.GetProtocolVersion() == protocol.VersionNQ
}
