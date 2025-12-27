# ✅ CORRECT Setup - Don't Delete Anything!

## The Problem
You already have a `TorrentSeeker` folder with all the Swift files inside. Xcode is asking to delete it because it wants to create a new one.

## The Solution - Two Options

### Option 1: Let Xcode Create in a Different Location (Recommended)

1. **Open Xcode**
2. **Create New Project** → iOS App → SwiftUI + Swift
3. **Name it**: `TorrentSeeker`
4. **SAVE TO**: `/Users/jeff/Desktop/` (temporary location)
5. Click **Create**

Now Xcode has created a proper project. Let's move the files:

6. **Close Xcode** completely
7. **Open Terminal** and run:
   ```bash
   # Copy the Xcode project file to the correct location
   cp -r ~/Desktop/TorrentSeeker/TorrentSeeker.xcodeproj /Users/jeff/Documents/WWW2020/torrent-seeker/ios-app/TorrentSeeker/

   # Delete the temporary project on Desktop
   rm -rf ~/Desktop/TorrentSeeker
   ```

8. **Open the project**:
   ```bash
   open /Users/jeff/Documents/WWW2020/torrent-seeker/ios-app/TorrentSeeker/TorrentSeeker.xcodeproj
   ```

9. In Xcode, you'll see errors because the default files are missing. **That's OK!**

10. **Delete** the references to:
    - `ContentView.swift` (it will show in red) - Right click → Delete → Remove Reference
    - `TorrentSeekerApp.swift` (the default one) - Right click → Delete → Remove Reference

11. **Add your existing files**:
    - Right-click on `TorrentSeeker` (yellow folder) → **Add Files to "TorrentSeeker"...**
    - Navigate to `/Users/jeff/Documents/WWW2020/torrent-seeker/ios-app/TorrentSeeker/TorrentSeeker/`
    - **Select ALL the .swift files and folders** (Models, Views, ViewModels, Services, Utils, TorrentSeekerApp.swift)
    - ✅ Check "Copy items if needed" is **UNCHECKED** (files are already there!)
    - ✅ Check "Create groups" is selected
    - ✅ Check "TorrentSeeker" target is selected
    - Click **Add**

12. **Build** (Cmd + B) - should compile successfully!

13. **Run** (Cmd + R) - app should launch!

---

### Option 2: Start Fresh in a New Location (Cleaner)

If Option 1 seems confusing, do this instead:

1. **Rename the existing folder** to avoid confusion:
   ```bash
   mv /Users/jeff/Documents/WWW2020/torrent-seeker/ios-app/TorrentSeeker /Users/jeff/Documents/WWW2020/torrent-seeker/ios-app/TorrentSeeker_SourceFiles
   ```

2. **Now create the Xcode project**:
   - Open Xcode
   - Create New Project → iOS App → SwiftUI + Swift
   - Name: `TorrentSeeker`
   - Save to: `/Users/jeff/Documents/WWW2020/torrent-seeker/ios-app/`
   - Click Create (now it won't ask to delete anything!)

3. **Delete default files** in Xcode:
   - `ContentView.swift` → Delete → Move to Trash
   - Keep `TorrentSeekerApp.swift` (we'll replace it)

4. **Add the source files**:
   - Right-click on `TorrentSeeker` folder → **Add Files to "TorrentSeeker"...**
   - Navigate to `/Users/jeff/Documents/WWW2020/torrent-seeker/ios-app/TorrentSeeker_SourceFiles/TorrentSeeker/`
   - Select **all folders** (Models, Views, ViewModels, Services, Utils) and `TorrentSeekerApp.swift`
   - ✅ Check "Copy items if needed"
   - ✅ Check "Create groups"
   - ✅ Check "TorrentSeeker" target
   - Click **Add**

5. **Build and Run!**

6. **Optional cleanup** (after confirming app works):
   ```bash
   rm -rf /Users/jeff/Documents/WWW2020/torrent-seeker/ios-app/TorrentSeeker_SourceFiles
   ```

---

## Which Option Should You Choose?

- **Option 1**: If you're comfortable with Terminal commands (faster)
- **Option 2**: If you prefer Xcode GUI (safer, more visual)

Both will work perfectly! The key insight is: **Xcode needs to create the .xcodeproj file itself**, but we can then add your existing Swift files to it.

## After Setup

Don't forget to:
1. Update the backend URL in `Services/APIService.swift` line 31
2. Add HTTP permissions in Info.plist (Project Settings → Info → Add "App Transport Security Settings" → "Allow Arbitrary Loads" = YES)
3. Set minimum iOS version to 17.0

Let me know which option you'd like to try!
