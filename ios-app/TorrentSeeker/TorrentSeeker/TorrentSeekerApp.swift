//
//  TorrentSeekerApp.swift
//  TorrentSeeker
//
//  Main app entry point
//

import SwiftUI

@main
struct TorrentSeekerApp: App {
    @StateObject private var authViewModel = AuthViewModel()

    var body: some Scene {
        WindowGroup {
            if authViewModel.isAuthenticated {
                MainView()
                    .environmentObject(authViewModel)
            } else {
                LoginView()
                    .environmentObject(authViewModel)
            }
        }
    }
}
