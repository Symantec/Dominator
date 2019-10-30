package serverutil

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func newPerUserMethodLimiter(
	perUserMethodLimitsInput map[string]uint) *PerUserMethodLimiter {
	perUserMethodLimits := make(map[string]uint, len(perUserMethodLimitsInput))
	for method, limit := range perUserMethodLimitsInput {
		perUserMethodLimits[method] = limit
	}
	return &PerUserMethodLimiter{
		perUserMethodCounts: make(map[userMethodType]uint,
			len(perUserMethodLimits)),
		perUserMethodLimits: perUserMethodLimits,
	}
}

func (limiter *PerUserMethodLimiter) blockMethod(methodName string,
	authInfo *srpc.AuthInformation) (func(), error) {
	if authInfo.HaveMethodAccess {
		return nil, nil
	}
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()
	if limit := limiter.perUserMethodLimits[methodName]; limit < 1 {
		return nil, nil
	} else {
		userMethod := userMethodType{
			method:   methodName,
			username: authInfo.Username,
		}
		if count := limiter.perUserMethodCounts[userMethod]; count >= limit {
			return nil, fmt.Errorf("%s reached limit of %d calls for %s",
				authInfo.Username, limit, methodName)
		} else {
			limiter.perUserMethodCounts[userMethod] = count + 1
			return func() {
				limiter.mutex.Lock()
				defer limiter.mutex.Unlock()
				if count := limiter.perUserMethodCounts[userMethod]; count < 1 {
					panic(fmt.Sprintf("%s has no %s calls to release",
						authInfo.Username, methodName))
				} else {
					if count -= 1; count < 1 {
						delete(limiter.perUserMethodCounts, userMethod)
					} else {
						limiter.perUserMethodCounts[userMethod] = count
					}
				}
			}, nil
		}
	}
}
