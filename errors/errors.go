package errors

import "errors"

var ErrNoPubKeyCallback = errors.New("no PublicKeyCallback in server config")
var ErrNoCmdsCallbacks = errors.New("no CommandsCallbacks in server config")
var ErrEmptyPrivKeyPath = errors.New("empty PrivateKeyPath in server config")

var ErrInvalidEnvArgs = errors.New("invalid env arguments")
var ErrNoSessionChannel = errors.New("no session channel")
var ErrNotSessionChannel = errors.New("terminal requires session channel")
