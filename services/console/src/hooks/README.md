# useActionlint Hook

A React hook for integrating WASM actionlint validation into your React components. This hook provides a clean interface for validating GitHub Actions YAML workflows while handling loading states, errors, caching, and cleanup internally.

## Features

- 🚀 **WASM Integration**: Seamlessly loads and manages the actionlint WASM module
- ⚡ **Debounced Validation**: Configurable debouncing to avoid excessive validation calls
- 💾 **Smart Caching**: Caches validation results to improve performance
- 🔄 **Real-time Validation**: Supports both debounced and immediate validation
- 🛡️ **Error Handling**: Comprehensive error handling for both system and validation errors
- 🧹 **Automatic Cleanup**: Proper cleanup of timers and WASM resources
- 📊 **State Management**: Complete state management with loading and error states
- 🎯 **TypeScript Support**: Full TypeScript support with comprehensive type definitions

## Installation

Ensure the WASM files are available in your public directory:

- `/public/main.wasm` - The actionlint WASM module
- `/public/wasm_exec.js` - Go WASM runtime support

## Basic Usage

```tsx
import React, { useState } from "react";
import { useActionlint } from "../hooks/useActionlint";

function YAMLValidator() {
  const [yamlContent, setYamlContent] = useState("");

  const { state, validateYaml, isReady, hasErrors } = useActionlint();

  const handleContentChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newContent = e.target.value;
    setYamlContent(newContent);
    validateYaml(newContent); // Debounced validation
  };

  if (!isReady) {
    return <div>Loading validation engine...</div>;
  }

  return (
    <div>
      <textarea
        value={yamlContent}
        onChange={handleContentChange}
        placeholder="Enter your GitHub Actions YAML..."
      />

      {state.isLoading && <div>Validating...</div>}

      {hasErrors && (
        <div>
          <h3>Validation Errors:</h3>
          {state.errors.map((error, index) => (
            <div key={index}>
              <strong>
                Line {error.line}, Col {error.column}:
              </strong>{" "}
              {error.message}
            </div>
          ))}
        </div>
      )}

      {!hasErrors && !state.isLoading && isReady && (
        <div>✅ No validation errors!</div>
      )}
    </div>
  );
}
```

## Advanced Usage

```tsx
import React, { useState } from "react";
import { useActionlint } from "../hooks/useActionlint";

function AdvancedYAMLValidator() {
  const [yamlContent, setYamlContent] = useState("");

  const {
    state,
    validateYaml,
    validateImmediate,
    reset,
    clearCache,
    reinitialize,
    isReady,
    hasErrors,
    hasSystemError,
    cacheStats,
  } = useActionlint({
    debounceMs: 500, // 500ms debounce
    enableCache: true, // Enable result caching
    cacheTtl: 60000, // Cache for 1 minute
    maxCacheSize: 20, // Keep up to 20 cached results
    wasmPath: "/custom/path.wasm", // Custom WASM path
  });

  const handleContentChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newContent = e.target.value;
    setYamlContent(newContent);
    validateYaml(newContent); // Debounced validation
  };

  const handleValidateNow = () => {
    validateImmediate(yamlContent); // Skip debouncing
  };

  const handleReset = () => {
    reset(); // Clear validation state
  };

  const handleClearCache = () => {
    clearCache(); // Clear cached results
  };

  const handleReinitialize = () => {
    reinitialize(); // Reinitialize WASM module
  };

  return (
    <div>
      {/* Status indicators */}
      <div>
        <span>Status: {isReady ? "Ready" : "Initializing..."}</span>
        <span>
          Cache: {cacheStats.size}/{cacheStats.maxSize}
        </span>
      </div>

      {/* Controls */}
      <div>
        <button onClick={handleValidateNow} disabled={!isReady}>
          Validate Now
        </button>
        <button onClick={handleReset}>Reset</button>
        <button onClick={handleClearCache}>Clear Cache</button>
        <button onClick={handleReinitialize}>Reinitialize</button>
      </div>

      {/* Editor */}
      <textarea
        value={yamlContent}
        onChange={handleContentChange}
        placeholder="Enter your GitHub Actions YAML..."
      />

      {/* System errors */}
      {hasSystemError && (
        <div>
          <h3>System Error:</h3>
          <p>{state.error}</p>
        </div>
      )}

      {/* Validation state */}
      {state.isLoading && <div>🔄 Validating...</div>}

      {/* Validation results */}
      {hasErrors && (
        <div>
          <h3>Found {state.errors.length} validation errors:</h3>
          {state.errors.map((error, index) => (
            <div key={index}>
              <div>
                <span>
                  Line {error.line}, Col {error.column}
                </span>
                <span>{error.kind}</span>
              </div>
              <p>{error.message}</p>
            </div>
          ))}
        </div>
      )}

      {!hasErrors && !state.isLoading && isReady && (
        <div>✅ No validation errors found!</div>
      )}

      {/* Debug info */}
      <details>
        <summary>Debug Info</summary>
        <pre>{JSON.stringify(state, null, 2)}</pre>
      </details>
    </div>
  );
}
```

## API Reference

### Hook Options

```tsx
interface UseActionlintOptions {
  debounceMs?: number; // Debounce delay (default: 300ms)
  autoValidate?: boolean; // Auto-validate on change (default: true)
  wasmPath?: string; // WASM file path (default: '/main.wasm')
  wasmExecPath?: string; // wasm_exec.js path (default: '/wasm_exec.js')
  enableCache?: boolean; // Enable caching (default: true)
  cacheTtl?: number; // Cache TTL (default: 60000ms)
  maxCacheSize?: number; // Max cache entries (default: 10)
}
```

### Return Value

```tsx
interface UseActionlintReturn {
  // State
  state: ActionlintState;
  isReady: boolean;
  hasErrors: boolean;
  hasSystemError: boolean;
  cacheStats: CacheStats;

  // Actions
  validateYaml: (content: string, immediate?: boolean) => void;
  validateImmediate: (content: string) => void;
  reset: () => void;
  reinitialize: () => void;
  clearCache: () => void;
}
```

### State Interface

```tsx
interface ActionlintState {
  isLoading: boolean;
  isInitialized: boolean;
  errors: ActionlintError[];
  error: string | null;
  lastValidated: string | null;
  validatedAt: Date | null;
}
```

### Error Interface

```tsx
interface ActionlintError {
  kind: string; // Error category
  message: string; // Error description
  line: number; // Line number (1-based)
  column: number; // Column number (1-based)
}
```

## Performance Considerations

### Caching

The hook includes built-in caching to avoid re-validating identical content:

- **Cache Key**: Content hash
- **Cache TTL**: Configurable expiration time
- **LRU Eviction**: Oldest entries removed when cache is full
- **Memory Efficient**: Only stores validation results, not content

### Debouncing

Debouncing prevents excessive validation calls during rapid typing:

- **Default Delay**: 300ms
- **Configurable**: Set via `debounceMs` option
- **Smart**: Immediate validation for paste operations
- **Cancellable**: Pending validations cancelled on new input

### WASM Loading

The WASM module is loaded once and reused:

- **Lazy Loading**: Module loaded on first use
- **Error Recovery**: Automatic retry on failure
- **Cleanup**: Proper cleanup on component unmount
- **Browser Compatibility**: Fallback for Safari and older browsers

## Error Handling

The hook distinguishes between two types of errors:

1. **System Errors**: WASM loading, initialization, or runtime errors
2. **Validation Errors**: Errors found in the YAML content

```tsx
// Check for system errors
if (hasSystemError) {
  console.error("System error:", state.error);
}

// Check for validation errors
if (hasErrors) {
  console.log("Validation errors:", state.errors);
}
```

## Browser Compatibility

- **Modern Browsers**: Full WebAssembly support
- **Safari**: Fallback for missing `WebAssembly.instantiateStreaming`
- **Older Browsers**: Graceful degradation with error messages

## Best Practices

1. **Initialize Early**: Place the hook high in your component tree
2. **Handle Loading**: Always check `isReady` before validation
3. **Error Boundaries**: Wrap components using the hook in error boundaries
4. **Memory Management**: Clear cache periodically in long-running apps
5. **User Feedback**: Show loading states and progress indicators

## Troubleshooting

### WASM Not Loading

- Ensure `/public/main.wasm` and `/public/wasm_exec.js` exist
- Check browser console for network errors
- Verify MIME types are configured correctly

### Validation Not Working

- Check `isReady` state before calling validation functions
- Verify WASM module initialization completed successfully
- Look for system errors in `state.error`

### Performance Issues

- Increase `debounceMs` for slower devices
- Reduce `maxCacheSize` to limit memory usage
- Disable caching with `enableCache: false` if needed

### Memory Leaks

- The hook automatically cleans up on unmount
- Call `clearCache()` periodically in long-running apps
- Use `reset()` to clear validation state when appropriate

## Examples

See `/src/components/ActionlintExample.tsx` for a complete working example.

## License

This hook is part of the InfraLayer project and follows the same license terms.
