# CloudWorkstation GUI Testing Framework

This directory contains the testing framework for the CloudWorkstation GUI, which uses the Fyne library for cross-platform desktop applications.

## Testing Approach

The GUI testing framework focuses on several key areas:

1. **Component Tests**: Testing individual UI components in isolation
2. **Cross-Platform Rendering**: Ensuring UI works correctly across different platforms
3. **System Tray Integration**: Validating system tray functionality
4. **Responsive Layout**: Testing UI adaptability to different screen sizes
5. **Visual Validation**: Using snapshots to validate visual appearance

## Running Tests

To run the GUI tests, use the provided script:

```bash
./scripts/test_gui.sh
```

This script will:
- Run all GUI tests
- Generate a coverage report
- Output test results to `test_results/` directory

## Test Structure

### Mock Components

The testing framework uses mock implementations of key interfaces:

- `MockCloudWorkstationAPI`: Mocks the API client for testing without real backend
- `MockProfileManager`: Mocks profile management operations
- `MockDesktopApp`: Mocks desktop app functionality for system tray testing

### Test Data

Test data fixtures are located in the `testdata/` directory:
- `instance_card.xml`: XML definition for instance card visual testing
- `notification.xml`: XML definition for notification component testing
- `dashboard.xml`: XML definition for dashboard view testing

### Custom Test Components

The framework includes custom components for testing:

- `responsive/grid_layout.go`: A responsive grid layout for testing UI adaptability

## Platform-Specific Testing

Tests can be run on different platforms with different configurations:

- macOS (Light and Dark mode)
- Windows
- Linux
- HiDPI displays
- Mobile form factors

Environment variables can be set to test different configurations:

```bash
FYNE_THEME=light FYNE_SCALE=2.0 ./scripts/test_gui.sh  # Test HiDPI displays
```

## Test Categories

### UI Component Tests (`ui_components_test.go`)

Tests rendering and functionality of individual UI components:
- Instance cards
- Notification system
- Dashboard view

### System Tray Tests (`system_tray_test.go`)

Tests system tray menu and functionality:
- Menu structure
- Action handling
- Icon updates

### Cross-Platform Tests (`cross_platform_test.go`)

Tests UI rendering across different platforms:
- Different theme variants (Light/Dark)
- Different display densities
- Different screen sizes

### Responsive Layout Tests

Tests UI adaptability to different screen sizes:
- Desktop layouts
- Laptop layouts
- Mobile layouts

## CI Integration

In CI environments, tests that require a display will be automatically skipped.
The test script detects CI environments and adjusts testing behavior accordingly.

## Adding New Tests

When adding new tests:

1. Create test files in this directory following the naming pattern `*_test.go`
2. For visual tests, add XML definitions in the `testdata/` directory
3. Add mock implementations as needed
4. Update the README if new testing approaches are added