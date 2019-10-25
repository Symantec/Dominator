package serverutil

import (
	"sync"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

type PerUserMethodLimiter struct {
	mutex               sync.Mutex
	perUserMethodCounts map[userMethodType]uint
	perUserMethodLimits map[string]uint
}

type userMethodType struct {
	method   string
	username string
}

func NewPerUserMethodLimiter(
	perUserMethodLimits map[string]uint) *PerUserMethodLimiter {
	return newPerUserMethodLimiter(perUserMethodLimits)
}

func (limiter *PerUserMethodLimiter) BlockMethod(methodName string,
	authInfo *srpc.AuthInformation) (func(), error) {
	return limiter.blockMethod(methodName, authInfo)
}
