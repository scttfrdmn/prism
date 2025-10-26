# Prism TUI Guide

*Interactive terminal interface with keyboard navigation*

## What is the TUI?

TUI stands for "Terminal User Interface." It's a colorful, interactive way to use Prism without typing lots of commands. You can navigate with arrow keys and see everything on one screen!

## How to Start the TUI

Open your terminal and type:

```bash
prism tui
```

That's it! You'll see a colorful screen with tabs at the top.

## Finding Your Way Around

The TUI has several different screens (called "views") that you can switch between:

**Navigation Tabs**: Dashboard | Instances | Templates | Storage | Settings | Profiles

### How to Move Between Views

- Use the **Left Arrow** and **Right Arrow** keys to move between tabs
- Or press **Tab** to move forward and **Shift+Tab** to move backward
- Press **?** anytime to see available keyboard shortcuts

## The Dashboard View

**Dashboard Overview**

The Dashboard shows you:
- How many cloud computers you have running
- How much they cost per day
- Recent activities
- Quick actions you can take

**Keyboard shortcuts:**
- **r**: Refresh the dashboard
- **→**: Go to Instances view

## The Instances View

**Instance Management**

This is where you can see and manage all your cloud computers.

For each instance, you'll see:
- **Name**: What you named your cloud computer
- **Template**: What research environment it uses
- **State**: Whether it's running or stopped
- **IP Address**: Its internet address (when running)
- **Cost/Day**: How much it costs per day

**Keyboard shortcuts:**
- **↑/↓**: Move up and down the list
- **Enter**: Show more details about a selected instance
- **s**: Start a stopped instance
- **p**: Stop a running instance
- **c**: Connect to an instance
- **d**: Delete an instance (it will ask you to confirm first)
- **r**: Refresh the list
- **/**: Search for a specific instance

## The Templates View

**Template Selection**

This is where you can see all the available research environments and launch new cloud computers.

**Keyboard shortcuts:**
- **↑/↓**: Move up and down the list
- **Enter**: See more details about a template
- **l**: Launch a new cloud computer using the selected template
- **r**: Refresh the template list
- **/**: Search for a specific template

### Launching a New Cloud Computer

When you press **l** to launch a template, you'll see a form where you can:

1. Enter a name for your new cloud computer
2. Choose a size (XS, S, M, L, XL) - bigger sizes are more powerful but cost more
3. See advanced options (optional)

## The Storage View

**Storage Management**

This is where you can manage your storage volumes (places to save your files).

### EFS Volumes (Shared Storage)

These are special storage volumes that can be shared between multiple cloud computers.

**Keyboard shortcuts:**
- **↑/↓**: Move up and down the list
- **Enter**: See volume details
- **c**: Create a new volume
- **d**: Delete a volume
- **r**: Refresh the list

### EBS Storage (Computer Storage)

These are storage volumes attached to a specific cloud computer.

**Keyboard shortcuts:**
- **↑/↓**: Move up and down the list
- **Enter**: See storage details
- **c**: Create a new storage volume
- **a**: Attach a volume to a cloud computer
- **d**: Detach a volume
- **x**: Delete a volume
- **r**: Refresh the list

## The Settings View

**Settings Configuration**

Here you can change how Prism works.

**Settings you can change:**
- **AWS Profile**: Which AWS account to use
- **AWS Region**: Which part of the world to create your cloud computers in
- **Theme**: Choose between light and dark colors
- **Registry**: Where to find templates

**Keyboard shortcuts:**
- **Tab/Shift+Tab**: Move between settings
- **Enter**: Change a setting
- **↑/↓**: Select from available options
- **s**: Save your changes

## Changing Colors: Light and Dark Theme

Prism can use either dark colors (easier on the eyes at night) or light colors (better in bright rooms).

To switch between them:
1. Go to the Settings view
2. Select the Theme setting
3. Press **Enter** to switch between Dark and Light
4. Press **s** to save your choice

## Helpful Messages

The TUI will show you messages at the top of the screen:

- **Green messages**: Success! Something worked correctly
- **Red messages**: Error - something went wrong
- **Blue messages**: Just information for you to know

These messages will disappear after a few seconds.

## Searching for Things

In any list (Instances, Templates, Storage), you can search by:

1. Pressing **/** to start searching
2. Typing what you're looking for
3. Pressing **Esc** to cancel the search and show everything again

## What If the TUI Doesn't Look Right?

If the TUI doesn't display correctly:
- Try making your terminal window bigger
- Check that your terminal supports colors
- Make sure you have the latest version of Prism

## Getting Out of the TUI

When you're done using the TUI:
- Press **q** to quit
- Or press **Ctrl+C**

You'll go back to the regular command line.

## Need More Help?

If you need more help, press **?** while in the TUI to see all available keyboard shortcuts for the current view.

Or check out these resources:
- The Prism help command: `prism help`
- The Getting Started Guide
- Ask your research supervisor for help