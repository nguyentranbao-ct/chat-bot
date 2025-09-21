package ctxval_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/nguyentranbao-ct/chat-bot/pkg/ctxval"
	"github.com/stretchr/testify/assert"
)

func TestWithValue(t *testing.T) {
	t.Parallel()
	type testKey string
	type testValue string

	t.Run("Set and Get Value", func(t *testing.T) {
		ctx := t.Context()
		key := testKey("key1")
		value := testValue("value1")
		ctx = ctxval.Wrap(ctx)
		ctxval.Set(ctx, key, value)
		retrievedValue, _ := ctxval.Get[testKey, testValue](ctx, key)

		assert.Equal(t, value, retrievedValue, "The retrieved value should be equal to the set value")
	})

	t.Run("Overwrite Value", func(t *testing.T) {
		ctx := t.Context()
		key := testKey("key1")
		value1 := testValue("value1")
		value2 := testValue("value2")
		ctx = ctxval.Wrap(ctx)
		ctxval.Set(ctx, key, value1)
		ctxval.Set(ctx, key, value2)
		retrievedValue, _ := ctxval.Get[testKey, testValue](ctx, key)

		assert.Equal(t, value2, retrievedValue, "The retrieved value should be the last set value")
	})

	t.Run("Get Non-Existent Value", func(t *testing.T) {
		ctx := t.Context()
		key := testKey("key1")
		ctx = ctxval.Wrap(ctx)
		_, ok := ctxval.Get[testKey, testValue](ctx, key)

		assert.False(t, ok, "The value should not exist in the context")
	})

	t.Run("Set and Get Multiple Values", func(t *testing.T) {
		ctx := t.Context()
		key1 := testKey("key1")
		value1 := testValue("value1")
		key2 := testKey("key2")
		value2 := testValue("value2")
		ctx = ctxval.Wrap(ctx)
		ctxval.Set(ctx, key1, value1)
		ctxval.Set(ctx, key2, value2)
		retrievedValue1, _ := ctxval.Get[testKey, testValue](ctx, key1)
		retrievedValue2, _ := ctxval.Get[testKey, testValue](ctx, key2)

		assert.Equal(t, value1, retrievedValue1, "The retrieved value for key1 should be equal to the set value")
		assert.Equal(t, value2, retrievedValue2, "The retrieved value for key2 should be equal to the set value")
	})

	t.Run("Set and Get Different Types", func(t *testing.T) {
		ctx := t.Context()
		type intKey string
		type stringKey string
		keyInt := intKey("intKey")
		keyString := stringKey("stringKey")
		intValue := 42
		stringValue := "value"
		ctx = ctxval.Wrap(ctx)
		ctxval.Set(ctx, keyInt, intValue)
		ctxval.Set(ctx, keyString, stringValue)
		retrievedIntValue, _ := ctxval.Get[intKey, int](ctx, keyInt)
		retrievedStringValue, _ := ctxval.Get[stringKey, string](ctx, keyString)

		assert.Equal(t, intValue, retrievedIntValue, "The retrieved int value should be equal to the set value")
		assert.Equal(t, stringValue, retrievedStringValue, "The retrieved string value should be equal to the set value")
	})
}

func TestConcurrentOperations(t *testing.T) {
	t.Parallel()
	t.Run("Concurrent Set Operations", func(t *testing.T) {
		ctx := ctxval.Wrap(t.Context())
		const numGoroutines = 100
		const numOperations = 1000

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := range numGoroutines {
			go func(routineID int) {
				defer wg.Done()
				for j := range numOperations {
					key := fmt.Sprintf("key-%d-%d", routineID, j)
					value := fmt.Sprintf("value-%d-%d", routineID, j)
					ctxval.Set(ctx, key, value)
				}
			}(i)
		}
		wg.Wait()
	})

	t.Run("Concurrent Get Operations", func(t *testing.T) {
		ctx := ctxval.Wrap(t.Context())
		const numKeys = 100

		// Setup initial values
		for i := range numKeys {
			key := fmt.Sprintf("key-%d", i)
			value := fmt.Sprintf("value-%d", i)
			ctxval.Set(ctx, key, value)
		}

		const numGoroutines = 100
		const numOperations = 1000

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for range numGoroutines {
			go func() {
				defer wg.Done()
				for j := range numOperations {
					key := fmt.Sprintf("key-%d", j%numKeys)
					val, ok := ctxval.Get[string, string](ctx, key)
					if !ok {
						t.Error("Expected value not found")
					}
					expectedVal := fmt.Sprintf("value-%d", j%numKeys)
					if val != expectedVal {
						t.Errorf("Got %s, want %s", val, expectedVal)
					}
				}
			}()
		}
		wg.Wait()
	})

	t.Run("Concurrent Set and Get Operations", func(t *testing.T) {
		ctx := ctxval.Wrap(t.Context())
		const numGoroutines = 50
		const numOperations = 1000

		var wg sync.WaitGroup
		wg.Add(numGoroutines * 2) // For both readers and writers

		// Writers
		for i := range numGoroutines {
			go func(routineID int) {
				defer wg.Done()
				for j := range numOperations {
					key := fmt.Sprintf("key-%d", j%100)
					value := fmt.Sprintf("value-%d-%d", routineID, j)
					ctxval.Set(ctx, key, value)
				}
			}(i)
		}

		// Readers
		for range numGoroutines {
			go func() {
				defer wg.Done()
				for j := range numOperations {
					key := fmt.Sprintf("key-%d", j%100)
					_, ok := ctxval.Get[string, string](ctx, key) //nolint:staticcheck
					if !ok {
						// It's okay if some values are not found during concurrent operations
						continue
					}
				}
			}()
		}
		wg.Wait()
	})

	t.Run("Race Condition Check", func(t *testing.T) {
		ctx := ctxval.Wrap(t.Context())
		const key = "race-key"
		const numGoroutines = 100
		const numOperations = 1000

		var wg sync.WaitGroup
		wg.Add(numGoroutines * 2)

		// Multiple writers for the same key
		for i := range numGoroutines {
			go func(val int) {
				defer wg.Done()
				for range numOperations {
					ctxval.Set(ctx, key, val)
				}
			}(i)
		}

		// Multiple readers for the same key
		for range numGoroutines {
			go func() {
				defer wg.Done()
				for range numOperations {
					_, _ = ctxval.Get[string, int](ctx, key)
				}
			}()
		}
		wg.Wait()
	})
}
