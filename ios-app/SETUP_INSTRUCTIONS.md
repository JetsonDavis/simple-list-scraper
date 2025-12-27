# Xcode Project Setup Instructions

## Quick Fix - Create Project Properly in Xcode

The crash happened because the Xcode project file (.xcodeproj) was incomplete. Follow these steps to create it properly:

### Step 1: Create New Xcode Project

1. **Open Xcode**
2. Select **"Create New Project"** (or File → New → Project)
3. Choose **iOS** → **App** template
4. Click **Next**

### Step 2: Configure Project

Fill in these details:
- **Product Name**: `TorrentSeeker`
- **Team**: Select your team (or leave as "None")
- **Organization Identifier**: `com.yourcompany` (or your preference)
- **Bundle Identifier**: Will auto-generate as `com.yourcompany.TorrentSeeker`
- **Interface**: Select **SwiftUI** ⚠️ IMPORTANT
- **Language**: Select **Swift** ⚠️ IMPORTANT
- **Storage**: Leave as "None"
- **Include Tests**: Uncheck both boxes (optional)

Click **Next**, then choose `/Users/jeff/Documents/WWW2020/torrent-seeker/ios-app/` as the save location.

### Step 3: Delete Default Files

Xcode will create some default files. In the Project Navigator (left sidebar), **DELETE** these files:
- `ContentView.swift` (right-click → Delete → Move to Trash)
- Keep `TorrentSeekerApp.swift` (we'll replace its contents)

### Step 4: Create Folder Structure

In Xcode's Project Navigator, right-click on the `TorrentSeeker` folder (the yellow one with your app icon):

1. **New Group** → Name it `Models`
2. **New Group** → Name it `Views`
3. **New Group** → Name it `ViewModels`
4. **New Group** → Name it `Services`
5. **New Group** → Name it `Utils`

Your structure should look like:
```
TorrentSeeker/
├── Models/
├── Views/
├── ViewModels/
├── Services/
├── Utils/
└── TorrentSeekerApp.swift
```

### Step 5: Add Swift Files

Now add the Swift files I created. For each file:

#### A. Models/Models.swift
1. Right-click on the **Models** folder → **New File**
2. Choose **Swift File** → **Next**
3. Name it `Models.swift` → **Create**
4. Replace contents with: `/Users/jeff/Documents/WWW2020/torrent-seeker/ios-app/TorrentSeeker/TorrentSeeker/Models/Models.swift`

#### B. Services/APIService.swift
1. Right-click on **Services** folder → **New File**
2. Choose **Swift File** → Name it `APIService.swift`
3. Copy contents from: `ios-app/TorrentSeeker/TorrentSeeker/Services/APIService.swift`

#### C. Services/WebSocketService.swift
1. Right-click on **Services** folder → **New File**
2. Name it `WebSocketService.swift`
3. Copy contents from: `ios-app/TorrentSeeker/TorrentSeeker/Services/WebSocketService.swift`

#### D. ViewModels/AuthViewModel.swift
1. Right-click on **ViewModels** folder → **New File**
2. Name it `AuthViewModel.swift`
3. Copy contents from: `ios-app/TorrentSeeker/TorrentSeeker/ViewModels/AuthViewModel.swift`

#### E. Views Files (7 files)
Repeat for each view file in the **Views** folder:
1. `LoginView.swift`
2. `MainView.swift`
3. `ItemsView.swift`
4. `MatchesView.swift`
5. `URLsView.swift`
6. `LogsView.swift`

#### F. Utils/Colors.swift
1. Right-click on **Utils** folder → **New File**
2. Name it `Colors.swift`
3. Copy contents from: `ios-app/TorrentSeeker/TorrentSeeker/Utils/Colors.swift`

#### G. Replace TorrentSeekerApp.swift
1. Click on the existing `TorrentSeekerApp.swift` in the root
2. Replace its contents with: `ios-app/TorrentSeeker/TorrentSeeker/TorrentSeekerApp.swift`

### Step 6: Update Info.plist for HTTP Access

1. In Project Navigator, click on `TorrentSeeker` (the blue project icon at the top)
2. Select the `TorrentSeeker` target
3. Go to **Info** tab
4. Right-click in the list → **Add Row**
5. Add: `App Transport Security Settings` (Type: Dictionary)
6. Click the arrow to expand it
7. Right-click → **Add Row** → `Allow Arbitrary Loads` (Type: Boolean) → Set to **YES**

This allows HTTP connections to your local backend.

### Step 7: Update Backend URL

1. Open `Services/APIService.swift`
2. Find line 31: `private let baseURL = "http://localhost:8080/api"`
3. Change `localhost:8080` to your backend's IP address if not running locally
4. For iOS Simulator, use `http://localhost:8080/api`
5. For Physical Device, use your computer's IP like `http://192.168.1.100:8080/api`

### Step 8: Set Minimum iOS Version

1. Click on project (blue icon)
2. Select `TorrentSeeker` target
3. Go to **General** tab
4. Set **Minimum Deployments** to **iOS 17.0**

### Step 9: Build and Run

1. Select a simulator (e.g., iPhone 15 Pro) from the device menu
2. Press **Cmd + R** or click the ▶️ Play button
3. App should build and launch!

## Troubleshooting

### If you get "Cannot find type" errors:
- Make sure all files are added to the TorrentSeeker target
- Click each .swift file → File Inspector (right panel) → check "TorrentSeeker" under Target Membership

### If app crashes on launch:
- Check the Console (Cmd + Shift + C) for error messages
- Verify backend URL is correct
- Make sure backend is running

### If you see connection errors:
- Verify backend is running: `curl http://localhost:8080/api/items`
- Check Info.plist has "Allow Arbitrary Loads" = YES
- For physical device, use computer's local IP address, not localhost

## Quick Setup Alternative

If you prefer, I can create a complete Package.swift for Swift Package Manager instead, which is simpler. Let me know!
