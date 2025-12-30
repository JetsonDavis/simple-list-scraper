//
//  MainView.swift
//  TorrentSeeker
//
//  Main view with tabs matching the React frontend
//

import SwiftUI

enum Tab {
    case items, matches, urls, logs
}

struct MainView: View {
    @EnvironmentObject var authViewModel: AuthViewModel
    @StateObject private var webSocketService = WebSocketService()

    @State private var selectedTab: Tab = .items
    @State private var items: [Item] = []
    @State private var urls: [SiteURL] = []
    @State private var matches: [Match] = []
    @State private var logs: [Log] = []
    @State private var logsPage = 1
    @State private var logsTotalPages = 0
    @State private var triggering = false

    // Azure blue color matching CSS
    let azureBlue = Color(red: 0.0, green: 0.47, blue: 0.831) // #0078D4

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Text("Torrent Seeker")
                    .font(.system(size: 20, weight: .semibold))
                    .foregroundColor(.white)

                Spacer()

                if let username = authViewModel.username {
                    Text(username)
                        .font(.system(size: 12))
                        .foregroundColor(.white.opacity(0.9))
                }

                Button(action: logout) {
                    Text("Logout")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundColor(.white)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 6)
                        .background(Color(red: 0.863, green: 0.208, blue: 0.271))
                        .cornerRadius(4)
                }
                .padding(.leading, 12)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 16)
            .background(azureBlue)

            // Tabs
            HStack(spacing: 0) {
                TabButton(title: "Items", isActive: selectedTab == .items) {
                    selectedTab = .items
                }
                TabButton(title: "Matches", isActive: selectedTab == .matches) {
                    selectedTab = .matches
                }
                TabButton(title: "Sites", isActive: selectedTab == .urls) {
                    selectedTab = .urls
                }
                TabButton(title: "Logs", isActive: selectedTab == .logs) {
                    selectedTab = .logs
                }
            }
            .background(Color(red: 0.953, green: 0.949, blue: 0.945)) // #F3F2F1
            .overlay(
                Rectangle()
                    .frame(height: 2)
                    .foregroundColor(Color(red: 0.882, green: 0.882, blue: 0.882)),
                alignment: .bottom
            )

            // Tab content with scrolling
            ScrollView {
                LazyVStack(spacing: 0) {
                    switch selectedTab {
                    case .items:
                        ItemsView(items: $items)
                            .padding(.horizontal, 16)
                            .padding(.top, 12)
                            .padding(.bottom, 32)
                    case .matches:
                        MatchesView(matches: $matches)
                            .padding(.horizontal, 16)
                            .padding(.top, 12)
                            .padding(.bottom, 32)
                    case .urls:
                        URLsView(urls: $urls)
                            .padding(.horizontal, 16)
                            .padding(.top, 12)
                            .padding(.bottom, 32)
                    case .logs:
                        LogsView(logs: $logs, logsPage: $logsPage, logsTotalPages: $logsTotalPages)
                            .padding(.horizontal, 16)
                            .padding(.top, 12)
                            .padding(.bottom, 32)
                    }
                }
                .frame(maxWidth: .infinity)
            }
            .background(Color.white)
        }
        .background(Color.white)
        .task {
            await loadInitialData()
            setupWebSocket()
        }
        .onChange(of: selectedTab) { _, newTab in
            Task {
                await loadTabData(tab: newTab)
            }
        }
    }

    func loadInitialData() async {
        do {
            items = try await APIService.shared.getItems()
            urls = try await APIService.shared.getURLs()
        } catch APIError.unauthorized {
            authViewModel.logout()
        } catch {
            print("Error loading initial data: \(error)")
        }
    }

    func loadTabData(tab: Tab) async {
        do {
            switch tab {
            case .matches:
                matches = try await APIService.shared.getMatches()
            case .logs:
                let response = try await APIService.shared.getLogs(page: 1)
                logs = response.logs
                logsPage = response.page
                logsTotalPages = response.totalPages
            default:
                break
            }
        } catch APIError.unauthorized {
            authViewModel.logout()
        } catch {
            print("Error loading tab data: \(error)")
        }
    }

    func setupWebSocket() {
        guard let token = authViewModel.token else { return }

        webSocketService.onWorkerStatus = { status, message in
            if status == "running" {
                triggering = true
            } else if status == "completed" {
                triggering = false
            }
        }

        webSocketService.onNewMatch = { match in
            matches.insert(match, at: 0)
        }

        webSocketService.onNewLog = { log in
            logs.insert(log, at: 0)
        }

        webSocketService.connect(token: token, host: "99.104.116.11:8080")
    }

    func triggerWorker() {
        Task {
            do {
                let response = try await APIService.shared.triggerWorker()
                if response.status == "triggered" {
                    triggering = true
                }
            } catch {
                print("Error triggering worker: \(error)")
            }
        }
    }

    func testSMS() {
        Task {
            do {
                let message = try await APIService.shared.testSMS()
                print(message)
            } catch {
                print("Error sending test SMS: \(error)")
            }
        }
    }

    func logout() {
        webSocketService.disconnect()
        authViewModel.logout()
    }
}

struct TabButton: View {
    let title: String
    let isActive: Bool
    let action: () -> Void

    let azureBlue = Color(red: 0.0, green: 0.47, blue: 0.831)
    let textSecondary = Color(red: 0.376, green: 0.369, blue: 0.361)

    var body: some View {
        Button(action: action) {
            VStack(spacing: 0) {
                Text(title)
                    .font(.system(size: 14, weight: .medium))
                    .foregroundColor(isActive ? azureBlue : textSecondary)
                    .padding(.vertical, 14)
                    .frame(maxWidth: .infinity)

                Rectangle()
                    .frame(height: 3)
                    .foregroundColor(isActive ? azureBlue : Color.clear)
            }
        }
        .frame(maxWidth: .infinity)
        .background(isActive ? Color.white : Color.clear)
    }
}

#Preview {
    MainView()
        .environmentObject(AuthViewModel())
}
