package views

// HomeMenuItem represents a menu item on the home screen.
type HomeMenuItem struct {
	Title       string
	Description string
	Key         string
}

// HomeMenuItems returns the list of menu items for the home screen.
func HomeMenuItems() []HomeMenuItem {
	return []HomeMenuItem{
		{
			Title:       "Calculate Fibonacci",
			Description: "Calculate a single Fibonacci number",
			Key:         "c",
		},
		{
			Title:       "Compare Algorithms",
			Description: "Compare all algorithms side by side",
			Key:         "m",
		},
		{
			Title:       "Settings",
			Description: "Configure theme and defaults",
			Key:         "s",
		},
		{
			Title:       "Help",
			Description: "View keyboard shortcuts and usage",
			Key:         "?",
		},
		{
			Title:       "Quit",
			Description: "Exit the application",
			Key:         "q",
		},
	}
}
