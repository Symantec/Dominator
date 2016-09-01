package resourcepool

import (
	"sync"
	"testing"
)

type testPoolType struct {
	sync.Mutex
	pool         *Pool
	max          int
	resources    []*testResourceType
	numActive    int
	maxNumActive int
	numInUse     int
	maxNumInUse  int
}

func newTestPool(max uint, numResources uint) *testPoolType {
	testPool := &testPoolType{
		pool:      New(max),
		max:       int(max),
		resources: make([]*testResourceType, 0, numResources),
	}
	for count := 0; count < int(numResources); count++ {
		testResource := &testResourceType{
			testPool: testPool,
			resource: testPool.pool.Create(),
		}
		testPool.resources = append(testPool.resources, testResource)
	}
	return testPool
}

func (testPool *testPoolType) getNumActive() int {
	testPool.Lock()
	defer testPool.Unlock()
	return testPool.numActive
}

func (testPool *testPoolType) getMaxNumActive() int {
	testPool.Lock()
	defer testPool.Unlock()
	return testPool.maxNumActive
}

func (testPool *testPoolType) getNumInUse() int {
	testPool.Lock()
	defer testPool.Unlock()
	return testPool.numInUse
}

func (testPool *testPoolType) getMaxNumInUse() int {
	testPool.Lock()
	defer testPool.Unlock()
	return testPool.maxNumInUse
}

type testResourceType struct {
	testPool *testPoolType
	resource *Resource
	active   bool
}

func (testResource *testResourceType) get(wait bool) bool {
	if !testResource.resource.Get(wait) {
		return false
	}
	testPool := testResource.testPool
	if !testResource.active {
		testResource.resource.SetReleaseFunc(testResource.releaseCallback)
	}
	testPool.Lock()
	defer testPool.Unlock()
	if testPool.numInUse >= testPool.max {
		panic("numInUse exceeding capacity")
	}
	testPool.numInUse++
	if testPool.numInUse > testPool.maxNumInUse {
		testPool.maxNumInUse = testPool.numInUse
	}
	if !testResource.active {
		if testPool.numActive >= testPool.max {
			panic("Capacity exceeded")
		}
		testPool.numActive++
		if testPool.numActive > testPool.maxNumActive {
			testPool.maxNumActive = testPool.numActive
		}
	}
	testResource.active = true
	return true
}

func (testResource *testResourceType) put() {
	testPool := testResource.testPool
	testPool.Lock()
	if testResource.active {
		testPool.numInUse--
	}
	testPool.Unlock()
	testResource.resource.Put()
}

func (testResource *testResourceType) release() {
	testPool := testResource.testPool
	testPool.Lock()
	testPool.numInUse--
	testPool.Unlock()
	testResource.resource.Release()
}

func (testResource *testResourceType) releaseCallback() {
	testPool := testResource.testPool
	testPool.Lock()
	defer testPool.Unlock()
	testPool.numActive--
	if testResource.active {
		testResource.active = false
	} else {
		panic("Resource re-released")
	}
}

func TestGetPut(t *testing.T) {
	testPool := newTestPool(1, 1)
	testResource := testPool.resources[0]
	if !testResource.get(false) {
		t.Errorf("Get(): would have waited")
	}
	tmp := testPool.getNumInUse()
	if tmp != 1 {
		t.Errorf("numInUse = %v", tmp)
	}
	if !testResource.active {
		t.Errorf("Resource should not have been released")
	}
	testResource.put()
	if !testResource.active {
		t.Errorf("Resource should not have been released")
	}
	tmp = testPool.getNumInUse()
	if tmp != 0 {
		t.Errorf("numInUse = %v", tmp)
	}
	tmp = testPool.getNumActive()
	if tmp != 1 {
		t.Errorf("numActive = %v", tmp)
	}
	tmp = testPool.getMaxNumActive()
	if tmp != 1 {
		t.Errorf("maxNumActive = %v", tmp)
	}
}

func TestGetClosePut(t *testing.T) {
	testPool := newTestPool(1, 1)
	testResource := testPool.resources[0]
	if !testResource.get(false) {
		t.Errorf("Get(): would have waited")
	}
	tmp := testPool.getNumInUse()
	if tmp != 1 {
		t.Errorf("numInUse = %v", tmp)
	}
	if !testResource.active {
		t.Errorf("Resource should not have been released")
	}
	testResource.release()
	tmp = testPool.getNumInUse()
	if tmp != 0 {
		t.Errorf("numInUse = %v", tmp)
	}
	if testResource.active {
		t.Errorf("Resource should have been released")
	}
	tmp = testPool.getNumActive()
	if tmp != 0 {
		t.Errorf("numActive = %v", tmp)
	}
	testResource.put()
}

func TestGetPutPut(t *testing.T) {
	testPool := newTestPool(1, 1)
	testResource := testPool.resources[0]
	defer func() {
		if err := recover(); err == nil {
			t.Errorf("Multiple Put() did not panic")
		}
	}()
	testResource.get(true)
	testResource.put()
	testResource.put()
}

func (testPool *testPoolType) testConcurrent(t *testing.T, numCycles int,
	testFunc func(*testResourceType, int)) {
	finished := make(chan struct{}, len(testPool.resources))
	for _, resource := range testPool.resources {
		go func(r *testResourceType) {
			testFunc(r, numCycles)
			finished <- struct{}{}
		}(resource)
	}
	for range testPool.resources {
		<-finished
	}
	tmp := testPool.getNumInUse()
	if tmp != 0 {
		t.Errorf("numInUse = %v", tmp)
	}
	tmp = testPool.getMaxNumInUse()
	expected := testPool.max
	if len(testPool.resources) < expected {
		expected = len(testPool.resources)
	}
	if tmp > expected {
		t.Errorf("maxNumInUse = %v", tmp)
	}
	tmp = testPool.getNumActive()
	if tmp > testPool.max {
		t.Errorf("numActive = %v", tmp)
	}
	tmp = testPool.getMaxNumActive()
	if tmp > expected {
		t.Errorf("maxNumActive = %v", tmp)
	}
}

func testManyGetPut(resource *testResourceType, numCycles int) {
	for count := 0; count < numCycles; count++ {
		resource.get(true)
		resource.put()
	}
}

func testManyGetClosePut(resource *testResourceType, numCycles int) {
	for count := 0; count < numCycles; count++ {
		resource.get(true)
		resource.release()
		resource.put()
	}
}

func TestLoopOne(t *testing.T) {
	testPool := newTestPool(10, 1)
	testManyGetPut(testPool.resources[0], 11)
	testManyGetClosePut(testPool.resources[0], 11)
}

func TestOnlyGetClosePutLoopOne(t *testing.T) {
	testPool := newTestPool(10, 1)
	testManyGetClosePut(testPool.resources[0], 11)
}

func TestLoopOneUnderCapacity(t *testing.T) {
	testPool := newTestPool(10, 9)
	testPool.testConcurrent(t, 1001, testManyGetPut)
	testPool.testConcurrent(t, 1001, testManyGetClosePut)
}

func TestOnlyGetClosePutLoopOneUnderCapacity(t *testing.T) {
	testPool := newTestPool(10, 9)
	testPool.testConcurrent(t, 1001, testManyGetClosePut)
}

func TestLoopAtCapacity(t *testing.T) {
	testPool := newTestPool(10, 10)
	testPool.testConcurrent(t, 1001, testManyGetPut)
	testPool.testConcurrent(t, 1001, testManyGetClosePut)
}

func TestOnlyGetClosePutLoopAtCapacity(t *testing.T) {
	testPool := newTestPool(10, 10)
	testPool.testConcurrent(t, 1001, testManyGetClosePut)
}

func TestLoopOneOverCapacity(t *testing.T) {
	testPool := newTestPool(10, 11)
	testPool.testConcurrent(t, 1001, testManyGetPut)
	testPool.testConcurrent(t, 1001, testManyGetClosePut)
}

func TestOnlyGetClosePutLoopOneOverCapacity(t *testing.T) {
	testPool := newTestPool(10, 11)
	testPool.testConcurrent(t, 1001, testManyGetClosePut)
}

func TestLoopFarOverCapacity(t *testing.T) {
	testPool := newTestPool(10, 113)
	testPool.testConcurrent(t, 1, testManyGetPut)
	testPool.testConcurrent(t, 1001, testManyGetClosePut)
}

func TestOnlyGetClosePutLoopFarOverCapacity(t *testing.T) {
	testPool := newTestPool(10, 113)
	testPool.testConcurrent(t, 1001, testManyGetClosePut)
}
