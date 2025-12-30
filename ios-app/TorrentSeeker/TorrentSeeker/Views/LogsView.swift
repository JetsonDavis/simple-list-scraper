//
//  LogsView.swift
//  TorrentSeeker
//
//  Logs tab view with mobile-friendly card layout
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

            // Logs list with card layout
            ForEach(logs) { log in
                LogCard(log: log)
            }

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

struct LogCard: View {
    let log: Log

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            // Status badge
            HStack {
                Text(log.success ? "SUCCESS" : "FAILED")
                    .font(.system(size: 11, weight: .bold))
                    .foregroundColor(log.success ? Color(red: 0.082, green: 0.341, blue: 0.141) : Color(red: 0.447, green: 0.11, blue: 0.141))
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(log.success ? Color(red: 0.831, green: 0.929, blue: 0.855) : Color(red: 0.973, green: 0.843, blue: 0.855))
                    .cornerRadius(4)

                Spacer()
            }

            // Timestamp
            VStack(alignment: .leading, spacing: 2) {
                Text("TIMESTAMP")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))

                Text(formatTimestamp(log.timestamp))
                    .font(.system(size: 13))
                    .foregroundColor(.primary)
            }

            // Description
            VStack(alignment: .leading, spacing: 2) {
                Text("DESCRIPTION")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))

                Text(log.description)
                    .font(.system(size: 13))
                    .foregroundColor(.primary)
                    .fixedSize(horizontal: false, vertical: true)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.white)
        .cornerRadius(4)
        .overlay(
            RoundedRectangle(cornerRadius: 4)
                .stroke(Color(red: 0.882, green: 0.882, blue: 0.882), lineWidth: 1)
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
            Log(id: 2, timestamp: "2024-01-01T12:05:00Z", description: "Failed operation with a much longer description that should wrap around to multiple lines", success: false)
        ]),
        logsPage: .constant(1),
        logsTotalPages: .constant(1)
    )
}
