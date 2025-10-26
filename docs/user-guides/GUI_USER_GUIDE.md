# Prism GUI Guide

<p align="center">
  <img src="images/prism.png" alt="Prism Logo" width="200">
</p>

## Welcome to the Prism GUI!

The Prism GUI makes managing your research cloud computers easy - no more typing commands! Just click buttons and see everything visually.

## What is the GUI?

GUI stands for "Graphical User Interface." Instead of typing commands, you'll use:
- Mouse clicks
- Buttons
- Menus
- Windows

The GUI is perfect for visual learners and anyone who prefers using a mouse over typing commands.

## Main Features of the GUI

### Main Dashboard

The Dashboard shows you:
- All your running and stopped cloud computers
- Real-time cost information per instance
- Quick launch buttons for common templates
- System status and notifications
- Profile and AWS region information

**Key Features:**
- **Instance Overview**: See all your instances at a glance with status indicators
- **Cost Monitoring**: Real-time cost tracking with daily estimates
- **Quick Actions**: Launch, stop, hibernate, and connect with one click

### Template Gallery

Browse research environments with pictures and descriptions:
- Visual cards for each template
- Detailed descriptions of included software
- One-click launching
- Size selection sliders (XS to XL)

### Instance Manager

Manage all your cloud computers in one place:
- Start and stop with a single click
- Connect buttons that open connections automatically
- Color-coded status indicators
- Detailed cost and usage information

### Storage Manager

Manage your data storage visually:
- Create new storage volumes with size sliders
- Drag-and-drop to attach storage to computers
- Visual space usage indicators
- One-click backup options

## Cool GUI Features

### System Tray / Menu Bar
The GUI runs in your system tray or menu bar (top of your screen), so you can:
- See status at a glance
- Get notifications about your cloud computers
- Access Prism without opening a terminal
- Monitor costs continuously

### Profile Management
Manage different AWS accounts from one place:
- Switch between your personal AWS account and invited accounts
- See which profile you're currently using in the sidebar
- Add new personal profiles for your own AWS accounts
- Add invitation profiles when someone invites you to use their account
- Learn more in the [Multi-Profile Guide](MULTI_PROFILE_GUIDE.md)

### Automatic Updates
The GUI checks for updates and lets you know when:
- New Prism versions are available
- New templates have been added
- Your cloud computers need attention

### Dark and Light Themes
Choose the colors that work best for you:
- Dark theme for nighttime use
- Light theme for daytime use
- Automatic switching based on your computer's settings

## How to Use the GUI

1. **Starting the GUI**
   ```bash
   prism gui
   ```
   Or click the Prism icon in your applications menu.

2. **Launching a Cloud Computer**
   - Click "Launch New Instance"
   - Select a template from the gallery
   - Enter a name for your computer
   - Choose a size (XS to XL)
   - Click "Launch"

3. **Connecting to Your Computer**
   - Find your computer in the Instances list
   - Click "Connect"
   - Choose SSH (command line) or Web (browser)
   - Start working!

4. **Managing Your Computer**
   - Click "Stop" when you're done for the day
   - Click "Start" when you want to use it again
   - Click "Delete" when you're completely finished

## Switching Between AWS Profiles

The Prism GUI makes it easy to switch between different AWS accounts:

1. **See your current profile**
   - Look in the sidebar under "AWS Profile"
   - It shows the name and type (Personal or Invitation)

2. **Switch profiles**
   - Click the "Switch Profile" button in the sidebar, or
   - Go to Settings → Profile Management
   - Select the profile you want to use

3. **Add a new personal profile**
   - Go to Settings → Profile Management
   - Click "Add Personal Profile"
   - Fill in the profile information
   - Click "Submit"

4. **Add an invitation profile**
   - Go to Settings → Profile Management
   - Click "Add Invitation"
   - Enter the invitation token and other details
   - Click "Submit"

When you switch profiles, the GUI automatically refreshes to show the cloud computers in that AWS account.

## Get Help

If you need help with the GUI:

1. Check the documentation at [docs.prism.org](https://docs.prism.org)

2. Run the tests to check your setup:
   ```bash
   prism test
   ```

3. Visit the community forum at [community.prism.org](https://community.prism.org)

4. Report issues on GitHub at [github.com/scttfrdmn/prism/issues](https://github.com/scttfrdmn/prism/issues)