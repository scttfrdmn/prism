// CloudWorkstation GUI (cws-gui) - Desktop application for research environments.
//
// Refactored using SOLID principles for maintainability and extensibility.
// The new architecture separates concerns into focused packages:
//   - ui/dashboard: Dashboard view and logic
//   - ui/instances: Instance management interface  
//   - ui/templates: Template browsing and launching
//   - ui/types: Shared types and interfaces
//
// This design follows:
//   - Single Responsibility: Each package has one clear purpose
//   - Open/Closed: Easy to add new sections without modifying existing code
//   - Liskov Substitution: All sections implement the same Section interface
//   - Interface Segregation: Focused interfaces for specific needs
//   - Dependency Inversion: High-level modules depend on abstractions
package main

import (
	"log"
	"os"

	"github.com/scttfrdmn/cloudworkstation/cmd/cws-gui/internal/ui"
)

func main() {
	// Create and initialize GUI using SOLID architecture
	gui, err := ui.NewGUI()
	if err != nil {
		log.Printf("Failed to create GUI: %v", err)
		os.Exit(1)
	}
	
	// Run the application
	gui.Run()
}