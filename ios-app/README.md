# Torrent Seeker - iOS App

A native iOS application built with Swift and SwiftUI that mimics the Torrent Seeker React frontend exactly.

## Features

- **Authentication**: Login and registration with JWT token authentication
- **Items Management**: Add, edit, and delete search items
- **Sites Management**: Manage torrent scraping URLs with custom display names
- **Matches View**: View and manage torrent matches with magnet links
- **Logs**: View operation logs with pagination
- **Real-time Updates**: WebSocket integration for live match and log updates
- **Worker Control**: Trigger scraping worker and monitor status

## Architecture

The app follows the MVVM (Model-View-ViewModel) pattern:

```
TorrentSeeker/
├── Models/
│   └── Models.swift              # Data models (Item, URL, Match, Log, etc.)
├── Views/
│   ├── LoginView.swift           # Login/Registration screen
│   ├── MainView.swift            # Main container with tabs
│   ├── ItemsView.swift           # Items management tab
│   ├── MatchesView.swift         # Torrent matches tab
│   ├── URLsView.swift            # Sites management tab
│   └── LogsView.swift            # Logs display tab
├── ViewModels/
│   └── AuthViewModel.swift       # Authentication state management
├── Services/
│   ├── APIService.swift          # REST API communication
│   └── WebSocketService.swift   # WebSocket real-time updates
├── Utils/
│   └── Colors.swift              # Color constants matching CSS
└── TorrentSeekerApp.swift        # Main app entry point
```

## Design

The UI precisely matches the React frontend design:

- **Azure Blue Theme**: Uses Microsoft Azure's color palette (#0078D4)
- **Gradient Login**: Purple-to-blue gradient background
- **Tabbed Interface**: Items, Matches, Sites, and Logs tabs
- **Table Views**: Data displayed in tables with inline editing
- **Real-time Feedback**: Loading spinners and status updates

## Setup

1. Open Xcode (version 15.0 or later)
2. Create a new iOS App project named "TorrentSeeker"
3. Set the minimum iOS version to 17.0
4. Copy all Swift files into the project
5. Update the backend URL in `APIService.swift`:

```swift
private let baseURL = "http://your-backend-url:8080/api"
```

6. For WebSocket connections, update the host in `MainView.swift`:

```swift
webSocketService.connect(token: token, host: "your-backend-url:8080")
```

## API Integration

The app connects to the same backend API as the React frontend:

- **Auth**: `/api/auth/login`, `/api/auth/register`
- **Items**: `/api/items` (GET, POST, PUT, DELETE)
- **URLs**: `/api/urls` (GET, POST, PUT, DELETE)
- **Matches**: `/api/matches` (GET, DELETE)
- **Logs**: `/api/logs` (GET, DELETE)
- **Worker**: `/api/trigger-worker` (POST)
- **WebSocket**: `/api/ws?token={token}`

## Requirements

- iOS 17.0+
- Xcode 15.0+
- Swift 5.9+
- SwiftUI

## Features Matching Frontend

### Login Screen
- Toggle between Login/Register
- Form validation
- Error display
- Gradient background
- Rounded white card design

### Main Interface
- Azure blue header with app title
- Username display
- Test SMS button
- Run worker button with spinner
- Logout button
- Tab navigation (Items, Matches, Sites, Logs)

### Items Tab
- Add new items
- Inline editing of existing items
- Delete with trash icon
- Real-time API sync

### Matches Tab
- Display torrent matches in table
- Clickable torrent links
- Magnet link icons
- Soft delete (hide) with eye-slash icon
- Hard delete with trash icon
- File size display
- Timestamp display

### Sites Tab
- Add new scraping URLs
- Edit display names
- Edit URLs
- Delete sites

### Logs Tab
- Paginated log display
- Success/Failed status badges
- Timestamp formatting
- Clear all logs button
- Previous/Next pagination

## WebSocket Updates

Real-time updates are received for:
- Worker status changes (running/completed)
- New matches added
- New logs created

## Notes

- Authentication tokens are stored in UserDefaults
- All API calls include automatic auth token handling
- Unauthorized (401) responses automatically log out the user
- The UI uses native iOS components while maintaining the original design aesthetic

## Building and Running

1. Open `TorrentSeeker.xcodeproj` in Xcode
2. Select a simulator or connected device
3. Press `Cmd + R` to build and run
4. Ensure your backend server is running and accessible

## Testing

The app includes SwiftUI previews for quick UI iteration:

```swift
#Preview {
    LoginView()
        .environmentObject(AuthViewModel())
}
```

Use Xcode's preview canvas to view and test individual views during development.
