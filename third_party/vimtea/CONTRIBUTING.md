# Contributing to VimTea

Thank you for your interest in contributing to VimTea! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for everyone.

## How to Contribute

### Reporting Bugs

If you find a bug, please create an issue on the GitHub repository with:

1. A clear, descriptive title
2. A detailed description of the bug
3. Steps to reproduce the behavior
4. Expected behavior
5. Screenshots if applicable
6. Your environment (OS, Go version, etc.)

### Suggesting Enhancements

Feature requests are welcome! Please create an issue with:

1. A clear, descriptive title
2. A detailed description of the proposed feature
3. Any relevant examples or use cases
4. If possible, a sketch of how the implementation might work

### Pull Requests

1. Fork the repository
2. Create a new branch for your changes
3. Make your changes
4. Run tests and make sure they pass
5. Submit a pull request to the main repository

When submitting a pull request, please:

- Include a clear description of the changes
- Link to any relevant issues
- Follow the existing code style
- Include tests for new functionality
- Update documentation as needed

## Development Setup

1. Clone the repository:
   ```
   git clone https://github.com/kujtimiihoxha/vimtea.git
   cd vimtea
   ```

2. Install dependencies:
   ```
   go mod download
   ```

3. Run the example:
   ```
   cd example
   go run main.go
   ```

## Code Style

- Follow standard Go code style and conventions
- Use `go fmt` before committing
- Use `goimports` to organize imports
- Write descriptive comments for exported functions
- Follow existing patterns in the codebase

## Testing

Add tests for new functionality. Run the tests with:

```
go test ./...
```

## Documentation

Update documentation when adding or changing features:

- Update godoc comments for all exported types, constants, variables, and functions
- Add detailed package documentation where needed
- Add explanatory comments for complex logic
- Update the README.md with examples if necessary
- Document new options or functions
- If adding a new option to the `options` struct, provide a corresponding `With*` function

Documentation should follow Go's best practices:
- Use complete sentences for package, type, and function comments
- Document all exported identifiers
- Use clear, concise language
- Explain "why" not just "what" for complex operations

## Project Structure

- **vimtea.go**: Package documentation and public API overview
- **model.go**: Main editor model and public interfaces
- **buffer.go**: Text buffer with undo/redo operations
- **cursor.go**: Cursor and text range operations
- **bindings.go**: Key binding registry
- **commands.go**: Command implementations
- **view.go**: Rendering functions
- **highlight.go**: Syntax highlighting
- **styles.go**: UI style definitions
- **wrapped_buffer.go**: Buffer interface adapter

## Questions?

If you have any questions about contributing, feel free to open an issue asking for clarification.

Thank you for your contributions!