// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"sync"
)

type VariablesDirectoryReader struct {
	ReadStub        func(path string) ([]interface{}, error)
	readMutex       sync.RWMutex
	readArgsForCall []struct {
		path string
	}
	readReturns struct {
		result1 []interface{}
		result2 error
	}
	readReturnsOnCall map[int]struct {
		result1 []interface{}
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *VariablesDirectoryReader) Read(path string) ([]interface{}, error) {
	fake.readMutex.Lock()
	ret, specificReturn := fake.readReturnsOnCall[len(fake.readArgsForCall)]
	fake.readArgsForCall = append(fake.readArgsForCall, struct {
		path string
	}{path})
	fake.recordInvocation("Read", []interface{}{path})
	fake.readMutex.Unlock()
	if fake.ReadStub != nil {
		return fake.ReadStub(path)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.readReturns.result1, fake.readReturns.result2
}

func (fake *VariablesDirectoryReader) ReadCallCount() int {
	fake.readMutex.RLock()
	defer fake.readMutex.RUnlock()
	return len(fake.readArgsForCall)
}

func (fake *VariablesDirectoryReader) ReadArgsForCall(i int) string {
	fake.readMutex.RLock()
	defer fake.readMutex.RUnlock()
	return fake.readArgsForCall[i].path
}

func (fake *VariablesDirectoryReader) ReadReturns(result1 []interface{}, result2 error) {
	fake.ReadStub = nil
	fake.readReturns = struct {
		result1 []interface{}
		result2 error
	}{result1, result2}
}

func (fake *VariablesDirectoryReader) ReadReturnsOnCall(i int, result1 []interface{}, result2 error) {
	fake.ReadStub = nil
	if fake.readReturnsOnCall == nil {
		fake.readReturnsOnCall = make(map[int]struct {
			result1 []interface{}
			result2 error
		})
	}
	fake.readReturnsOnCall[i] = struct {
		result1 []interface{}
		result2 error
	}{result1, result2}
}

func (fake *VariablesDirectoryReader) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.readMutex.RLock()
	defer fake.readMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *VariablesDirectoryReader) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}