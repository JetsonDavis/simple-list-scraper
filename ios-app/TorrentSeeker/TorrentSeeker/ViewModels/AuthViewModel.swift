//
//  AuthViewModel.swift
//  TorrentSeeker
//
//  Authentication state management
//

import Foundation
import SwiftUI

class AuthViewModel: ObservableObject {
    @Published var isAuthenticated = false
    @Published var username: String?
    @Published var token: String?

    init() {
        // Load saved auth state
        if let savedToken = UserDefaults.standard.string(forKey: "authToken"),
           let savedUsername = UserDefaults.standard.string(forKey: "username") {
            self.token = savedToken
            self.username = savedUsername
            self.isAuthenticated = true
        }
    }

    func login(token: String, username: String) {
        self.token = token
        self.username = username
        self.isAuthenticated = true

        UserDefaults.standard.set(token, forKey: "authToken")
        UserDefaults.standard.set(username, forKey: "username")
    }

    func logout() {
        self.token = nil
        self.username = nil
        self.isAuthenticated = false

        UserDefaults.standard.removeObject(forKey: "authToken")
        UserDefaults.standard.removeObject(forKey: "username")
    }
}
