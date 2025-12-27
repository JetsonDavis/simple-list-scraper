//
//  LogsView.swift
//  TorrentSeeker
//
//  Logs tab view matching the React frontend
//

import SwiftUI

struct LogsView: View {
    @Binding var logs: [Log]
    @Binding var logsPage: Int
    @Binding var logsTotalPages: Int

    let borderColor = Color(red: 0.882, green: 0.882, blue: 0.882)

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            // Clear logs button
            HStack {
                Spacer()
                Button(action: clearLogs) {
                    Text("Clear Logs")
                        .font(.system(size: 14, weight: .medium))
                        .foregroundColor(.white)
                        .padding(.horizontal, 20)
                        .padding(.vertical, 10)
                        .background(Color(red: 0.0, green: 0.47, blue: 0.831))
                        .cornerRadius(4)
                }
            }

            // Logs table
            VStack(spacing: 0) {
                // Table header
                HStack {
                    Text("TIMESTAMP")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))
                        .frame(width: 180, alignment: .leading)
                        .textCase(.uppercase)

                    Text("DESCRIPTION")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .textCase(.uppercase)

                    Text("STATUS")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))
                        .frame(width: 100, alignment: .center)
                        .textCase(.uppercase)
                }
                .padding(12)
                .background(Color(red: 0.953, green: 0.949, blue: 0.945))
                .overlay(
                    Rectangle()
                        .frame(height: 2)
                        .foregroundColor(borderColor),
                    alignment: .bottom
                )

                // Table rows
                ScrollView {
                    LazyVStack(spacing: 0) {
                        ForEach(logs) { log in
                            LogRow(log: log)
                        }
                    }
                }
            }
            .background(Color.white)
            .cornerRadius(4)
            .overlay(
                RoundedRectangle(cornerRadius: 4)
                    .stroke(borderColor, lineWidth: 1)
            )

            // Pagination
            if logsTotalPages > 1 {
                HStack(spacing: 10) {
                    Button(action: { loadLogs(page: logsPage - 1) }) {
                        Text("Previous")
                            .font(.system(size: 14, weight: .medium))
                            .foregroundColor(.white)
                            .padding(.horizontal, 16)
                            .padding(.vertical, 8)
                            .background(logsPage == 1 ? Color.gray : Color(red: 0.0, green: 0.47, blue: 0.831))
                            .cornerRadius(4)
                    }
                    .disabled(logsPage == 1)

                    Text("Page \(logsPage) of \(logsTotalPages)")
                        .font(.system(size: 14))

                    Button(action: { loadLogs(page: logsPage + 1) }) {
                        Text("Next")
                            .font(.system(size: 14, weight: .medium))
                            .foregroundColor(.white)
                            .padding(.horizontal, 16)
                            .padding(.vertical, 8)
                            .background(logsPage == logsTotalPages ? Color.gray : Color(red: 0.0, green: 0.47, blue: 0.831))
                            .cornerRadius(4)
                    }
                    .disabled(logsPage == logsTotalPages)
                }
                .frame(maxWidth: .infinity)
            }
        }
    }

    func clearLogs() {
        Task {
            do {
                try await APIService.shared.clearLogs()
                logs = []
                logsPage = 1
                logsTotalPages = 0
            } catch {
                print("Error clearing logs: \(error)")
            }
        }
    }

    func loadLogs(page: Int) {
        Task {
            do {
                let response = try await APIService.shared.getLogs(page: page)
                logs = response.logs
                logsPage = response.page
                logsTotalPages = response.totalPages
            } catch {
                print("Error loading logs: \(error)")
            }
        }
    }
}

struct LogRow: View {
    let log: Log

    var body: some View {
        HStack {
            // Timestamp
            Text(formatTimestamp(log.timestamp))
                .font(.system(size: 13))
                .frame(width: 180, alignment: .leading)

            // Description
            Text(log.description)
                .font(.system(size: 14))
                .frame(maxWidth: .infinity, alignment: .leading)

            // Status badge
            Text(log.success ? "SUCCESS" : "FAILED")
                .font(.system(size: 12, weight: .bold))
                .foregroundColor(log.success ? Color(red: 0.082, green: 0.341, blue: 0.141) : Color(red: 0.447, green: 0.11, blue: 0.141))
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(log.success ? Color(red: 0.831, green: 0.929, blue: 0.855) : Color(red: 0.973, green: 0.843, blue: 0.855))
                .cornerRadius(4)
                .frame(width: 100, alignment: .center)
        }
        .padding(.vertical, 8)
        .padding(.horizontal, 12)
        .background(Color.white)
        .overlay(
            Rectangle()
                .frame(height: 1)
                .foregroundColor(Color(red: 0.953, green: 0.949, blue: 0.945)),
            alignment: .bottom
        )
    }

    func formatTimestamp(_ timestamp: String) -> String {
        // Parse ISO timestamp and format to locale string
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]

        if let date = formatter.date(from: timestamp) {
            let displayFormatter = DateFormatter()
            displayFormatter.dateStyle = .short
            displayFormatter.timeStyle = .medium
            return displayFormatter.string(from: date)
        }

        return timestamp
    }
}

#Preview {
    LogsView(
        logs: .constant([
            Log(id: 1, timestamp: "2024-01-01T12:00:00Z", description: "Test log entry", success: true),
            Log(id: 2, timestamp: "2024-01-01T12:05:00Z", description: "Failed operation", success: false)
        ]),
        logsPage: .constant(1),
        logsTotalPages: .constant(1)
    )
}
