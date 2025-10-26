# Prism GUI Troubleshooting Guide

**Version**: 0.5.1
**Last Updated**: October 15, 2025

---

## Common Issues and Solutions

### Issue: GUI Shows 0 Templates or Empty Data

**Symptoms**:
- Dashboard shows "Available Templates: 0"
- Instance list is empty (when you know instances exist)
- All data views show 0 counts
- GUI shows "Connected" status but no data loads

**Root Cause**:
The Prism daemon (`cwsd`) was not running when the GUI attempted to load data.

**Solution (v0.5.2+)**: ✅ **Auto-Fixed!**

Starting in v0.5.2, the GUI **automatically starts the daemon** if it's not running. You should see:
```
🔍 Checking if daemon is running...
⚠️  Daemon is not running, attempting to start...
📍 Found daemon at: /path/to/cwsd
⏳ Waiting for daemon to initialize...
✅ Daemon started successfully!
```

The GUI will:
1. Check if daemon is responding (health check on port 8947)
2. If not, locate the `cwsd` binary automatically
3. Start daemon in independent process group
4. Wait up to 10 seconds for daemon to become healthy
5. Proceed with GUI startup

**If auto-start fails**:

1. **Check daemon binary exists**:
   ```bash
   ls -l bin/cwsd
   ```

2. **Manually start daemon**:
   ```bash
   ./bin/cws daemon start
   ```

3. **Check for errors in GUI console output**:
   - Failed to find daemon binary
   - Permission denied executing daemon
   - Daemon started but didn't become healthy

4. **Refresh GUI data** after manually starting daemon:
   - Click "Refresh" button
   - Or press Cmd/Ctrl+R

**Legacy Behavior (pre-v0.5.2)**:
- Daemon had to be started manually before launching GUI
- Use: `./bin/cws daemon start && ./bin/cws-gui`

---

### Issue: GUI Shows "Connection: Disconnected"

**Symptoms**:
- Red "Disconnected" indicator in System Status
- "Test Connection" button available
- No data loads in any view

**Root Cause**:
The GUI cannot connect to the daemon API on `http://localhost:8947`.

**Solution**:

1. **Check if daemon is running**:
   ```bash
   ./bin/cws daemon status
   ```

2. **Check if port 8947 is available**:
   ```bash
   lsof -i :8947
   ```

   Should show `cwsd` process listening on port 8947.

3. **Verify daemon API responds**:
   ```bash
   curl http://localhost:8947/api/v1/health
   ```

   Should return health status JSON.

4. **Restart daemon if needed**:
   ```bash
   ./bin/cws daemon stop
   ./bin/cws daemon start
   ```

5. **Click "Test Connection" in GUI** or refresh browser.

**If problem persists**:
- Check firewall settings (allow localhost:8947)
- Check for port conflicts (another process using 8947)
- Review daemon logs for errors

---

### Issue: Templates Load But Show Incorrect Count

**Symptoms**:
- GUI shows different template count than CLI
- Some templates missing from GUI view

**Root Cause**:
- Stale cache or state in GUI
- Template validation errors in frontend
- Backend/frontend sync issue

**Solution**:

1. **Compare CLI vs API counts**:
   ```bash
   # CLI count
   ./bin/cws templates | grep "Available Templates"

   # API count
   curl -s http://localhost:8947/api/v1/templates | jq 'keys | length'
   ```

2. **If counts differ**:
   - Restart daemon: `./bin/cws daemon restart`
   - Clear browser cache (if GUI is web-based)
   - Reload GUI application

3. **Check browser console** for JavaScript errors:
   - Open Developer Tools (Cmd+Option+I on Mac)
   - Look for errors in Console tab
   - Check Network tab for failed API requests

---

### Issue: Keyboard Shortcuts Not Working

**Symptoms**:
- Pressing Cmd/Ctrl+R doesn't refresh
- Number keys don't navigate views
- ? doesn't show help

**Root Cause**:
- Focus is in an input field (shortcuts intentionally disabled)
- Browser shortcuts override application shortcuts
- Keyboard event handler not registered

**Solution**:

1. **Click outside input fields**:
   - Shortcuts are disabled when typing in inputs
   - Click on empty area or press Escape

2. **Check browser shortcuts**:
   - Some browser shortcuts take precedence
   - Try using application in fullscreen mode

3. **Verify shortcuts are enabled**:
   - Open browser console
   - Look for "keydown" event listeners
   - Check for JavaScript errors on load

**Available Shortcuts**:
- **Cmd/Ctrl+R**: Refresh data
- **Cmd/Ctrl+K**: Focus search/filter
- **1**: Navigate to Dashboard
- **2**: Navigate to Templates
- **3**: Navigate to Instances
- **4**: Navigate to Storage
- **5**: Navigate to Projects
- **6**: Navigate to Users
- **7**: Navigate to Settings
- **?**: Show keyboard shortcuts help

---

### Issue: Bulk Operations Not Working

**Symptoms**:
- Checkboxes don't appear on instances table
- Can't select multiple instances
- Bulk action buttons not visible

**Root Cause**:
- TypeScript errors in build
- PropertyFilter component conflict
- State management issue

**Solution**:

1. **Verify build is clean**:
   ```bash
   cd cmd/cws-gui/frontend
   npm run build
   ```

   Should complete with 0 errors.

2. **Check browser console** for errors when clicking checkboxes.

3. **Verify instances are loaded**:
   - Must have instances in "My Instances" view
   - Empty table shows no checkboxes by design

4. **Try selecting single instance**:
   - Click on table row
   - Checkbox should appear on left side
   - Bulk actions toolbar should appear

---

### Issue: Advanced Filtering Not Working

**Symptoms**:
- PropertyFilter input doesn't show
- Search doesn't filter results
- Filtering properties dropdown empty

**Root Cause**:
- Cloudscape PropertyFilter not loaded
- JavaScript bundle loading error
- Filter state not initialized

**Solution**:

1. **Verify Cloudscape bundle loaded**:
   - Open Network tab in Developer Tools
   - Look for `cloudscape-*.js` file
   - Should be ~665KB and load successfully

2. **Check filter state initialization**:
   ```javascript
   // In browser console
   console.log('PropertyFilter loaded:', typeof PropertyFilter !== 'undefined');
   ```

3. **Try free-text search first**:
   - Type in filter input (e.g., "test")
   - Should filter across all fields
   - If this works, property-specific filtering should too

4. **Rebuild frontend**:
   ```bash
   cd cmd/cws-gui/frontend
   npm install
   npm run build
   cd ../../..
   make build
   ```

---

### Issue: Onboarding Wizard Shows Every Time

**Symptoms**:
- 3-step onboarding wizard appears on every launch
- "Skip" doesn't persist
- Can't dismiss permanently

**Root Cause**:
- localStorage not working or cleared
- Browser privacy mode blocking localStorage
- localStorage quota exceeded

**Solution**:

1. **Check localStorage support**:
   ```javascript
   // In browser console
   console.log('localStorage works:', typeof localStorage !== 'undefined');
   ```

2. **Manually mark onboarding complete**:
   ```javascript
   // In browser console
   localStorage.setItem('cws_onboarding_complete', 'true');
   ```

   Then reload GUI.

3. **Check browser privacy settings**:
   - Not in Incognito/Private mode
   - localStorage not disabled in settings
   - No browser extension blocking storage

4. **Clear and reset localStorage**:
   ```javascript
   // In browser console
   localStorage.clear();
   localStorage.setItem('cws_onboarding_complete', 'true');
   ```

---

### Issue: Focus Indicators Not Visible

**Symptoms**:
- Can't see which element has keyboard focus
- Tab navigation doesn't show outline
- Focus styles not applied

**Root Cause**:
- CSS not loaded
- Browser high contrast mode interfering
- Focus-visible polyfill issue

**Solution**:

1. **Verify CSS loaded**:
   - Open Network tab in Developer Tools
   - Look for `main-*.css` files
   - Should load without errors

2. **Test with different browser**:
   - Some browsers have better :focus-visible support
   - Chrome/Edge recommended for best results

3. **Check forced colors mode**:
   - macOS: System Preferences > Accessibility > Display
   - Windows: Settings > Ease of Access > High contrast
   - Disable if enabled, may interfere with custom focus styles

4. **Inspect element focus**:
   ```javascript
   // In browser console
   document.activeElement
   ```

   Should show currently focused element.

---

### Issue: ARIA Labels Not Announced

**Symptoms**:
- Screen reader doesn't announce status indicators
- Form errors not read aloud
- Navigation unclear with screen reader

**Root Cause**:
- Screen reader not enabled
- ARIA attributes not applied
- Browser incompatibility

**Solution**:

1. **Verify screen reader is running**:
   - macOS: VoiceOver (Cmd+F5)
   - Windows: NVDA or JAWS
   - Test basic navigation first

2. **Check ARIA attributes in DOM**:
   ```javascript
   // In browser console
   document.querySelectorAll('[aria-label]').length
   ```

   Should show many elements (50+).

3. **Test specific elements**:
   - Navigate to status indicator
   - Should announce: "Status: running" or similar
   - If not, check element in Inspector

4. **Verify Cloudscape ARIA support**:
   - Cloudscape components include ARIA by default
   - Check Cloudscape version is 3.0+
   - Update if needed: `npm update @cloudscape-design/components`

---

## Performance Issues

### Issue: GUI Slow to Load

**Symptoms**:
- Takes >5 seconds to show interface
- White screen on startup
- Assets loading slowly

**Solution**:

1. **Check network latency**:
   - Even localhost should be fast (<100ms)
   - Check Network tab for slow requests

2. **Verify build optimization**:
   ```bash
   cd cmd/cws-gui/frontend
   npm run build
   ```

   Check bundle sizes:
   - Main: ~270KB (gzipped: ~77KB)
   - Cloudscape: ~665KB (gzipped: ~183KB)

   If much larger, may have development build.

3. **Check daemon performance**:
   ```bash
   time curl http://localhost:8947/api/v1/templates
   ```

   Should complete in <50ms.

4. **Disable browser extensions**:
   - Ad blockers may interfere
   - Developer tools may slow rendering
   - Try in clean browser profile

---

### Issue: High Memory Usage

**Symptoms**:
- GUI uses >500MB RAM
- Browser tab becomes unresponsive
- Computer fans spin up

**Solution**:

1. **Check for memory leaks**:
   - Open Developer Tools > Performance
   - Record memory profile
   - Look for increasing memory over time

2. **Verify React cleanup**:
   - useEffect hooks should return cleanup functions
   - Event listeners should be removed
   - Timers should be cleared

3. **Reload GUI periodically**:
   - Close and reopen if running for hours
   - Browser memory will reset

4. **Check daemon memory**:
   ```bash
   ps aux | grep cwsd
   ```

   Should be <200MB normally.

---

## Debugging Tools

### Enable Console Logging

The GUI includes comprehensive console logging for debugging:

```javascript
// Already enabled in development
console.log('Loading Prism data...');
console.error('Failed to fetch templates:', error);
```

**View logs**:
1. Open Developer Tools (Cmd+Option+I on Mac, F12 on Windows)
2. Click Console tab
3. Look for Prism messages
4. Errors shown in red

### Check API Requests

**View all API calls**:
1. Open Developer Tools > Network tab
2. Filter by "XHR" or "Fetch"
3. Look for requests to `localhost:8947`
4. Check status codes (should be 200)
5. Inspect request/response payloads

**Test API manually**:
```bash
# Templates
curl http://localhost:8947/api/v1/templates

# Instances
curl http://localhost:8947/api/v1/instances

# Health check
curl http://localhost:8947/api/v1/health
```

### React DevTools

**Install React DevTools**:
- Chrome: https://chrome.google.com/webstore (search "React Developer Tools")
- Firefox: https://addons.mozilla.org/en-US/firefox (search "React DevTools")

**Use to inspect**:
- Component tree
- Props and state
- Re-render performance
- Hook values

---

## Getting Help

### Before Reporting Issues

1. **Check daemon status**: `./bin/cws daemon status`
2. **Check browser console**: Look for JavaScript errors
3. **Check API directly**: Test endpoints with curl
4. **Try different browser**: Rule out browser-specific issues
5. **Check system resources**: Sufficient RAM, disk space

### Reporting Issues

**Include in bug reports**:
- Prism version: `./bin/cws version`
- Operating system and version
- Browser and version
- Daemon status output
- Browser console errors (screenshot or copy/paste)
- Steps to reproduce
- Expected vs actual behavior

**Where to report**:
- GitHub Issues: https://github.com/scttfrdmn/prism/issues
- Include "[GUI]" prefix in issue title

### Diagnostic Commands

Run these commands and include output in bug reports:

```bash
# Version info
./bin/cws version

# Daemon status
./bin/cws daemon status

# API health check
curl http://localhost:8947/api/v1/health

# Check port
lsof -i :8947

# Template count verification
./bin/cws templates | wc -l
curl -s http://localhost:8947/api/v1/templates | jq 'keys | length'

# System info
uname -a
```

---

## Quick Reference

### Essential Commands

```bash
# Start daemon
./bin/cws daemon start

# Stop daemon
./bin/cws daemon stop

# Check daemon status
./bin/cws daemon status

# Launch GUI
./bin/cws-gui

# Test API
curl http://localhost:8947/api/v1/health
```

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| Cmd/Ctrl+R | Refresh data |
| Cmd/Ctrl+K | Focus search |
| 1-7 | Navigate views |
| ? | Show help |

### Troubleshooting Checklist

- [ ] Daemon is running
- [ ] API responds on port 8947
- [ ] Browser console shows no errors
- [ ] Network tab shows successful requests
- [ ] localStorage is enabled
- [ ] Not in private/incognito mode
- [ ] Browser is up-to-date
- [ ] Sufficient system resources

---

**Last Updated**: October 15, 2025
**Version**: 0.5.1
**For More Help**: See PRODUCTION_READINESS_CHECKLIST.md and SPRINT_0-2_COMPLETION_SUMMARY.md
