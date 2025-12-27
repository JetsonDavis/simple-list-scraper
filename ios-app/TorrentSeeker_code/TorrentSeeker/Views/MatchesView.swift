//
//  MatchesView.swift
//  TorrentSeeker
//
//  Matches tab view matching the React frontend
//

import SwiftUI

struct MatchesView: View {
    @Binding var matches: [Match]

    let borderColor = Color(red: 0.882, green: 0.882, blue: 0.882)

    var body: some View {
        VStack(spacing: 0) {
            // Table header
            HStack(spacing: 8) {
                Text("ITEM")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))
                    .frame(width: 120, alignment: .leading)

                Text("SITE")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))
                    .frame(width: 80, alignment: .leading)

                Text("TORRENT TEXT")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))
                    .frame(maxWidth: .infinity, alignment: .leading)

                Text("SIZE")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))
                    .frame(width: 60, alignment: .center)

                Text("MAGNET")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))
                    .frame(width: 60, alignment: .center)

                Text("WHEN")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))
                    .frame(width: 100, alignment: .leading)

                Text("ACTIONS")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))
                    .frame(width: 80, alignment: .center)
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
                    ForEach(matches) { match in
                        MatchRow(match: match, onSoftDelete: {
                            softDeleteMatch(id: match.id)
                        }, onHardDelete: {
                            hardDeleteMatch(id: match.id)
                        })
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
    }

    func softDeleteMatch(id: Int) {
        Task {
            do {
                try await APIService.shared.softDeleteMatch(id: id)
                matches = try await APIService.shared.getMatches()
            } catch {
                print("Error soft deleting match: \(error)")
            }
        }
    }

    func hardDeleteMatch(id: Int) {
        Task {
            do {
                try await APIService.shared.hardDeleteMatch(id: id)
                matches = try await APIService.shared.getMatches()
            } catch {
                print("Error hard deleting match: \(error)")
            }
        }
    }
}

struct MatchRow: View {
    let match: Match
    let onSoftDelete: () -> Void
    let onHardDelete: () -> Void

    let azureBlue = Color(red: 0.0, green: 0.47, blue: 0.831)

    var body: some View {
        HStack(spacing: 8) {
            // Item
            Text(match.item)
                .font(.system(size: 14))
                .frame(width: 120, alignment: .leading)
                .lineLimit(2)

            // Site
            Text(match.site)
                .font(.system(size: 14))
                .frame(width: 80, alignment: .leading)
                .lineLimit(1)

            // Torrent text (with link)
            if let urlString = Foundation.URL(string: match.url) {
                Link(destination: urlString) {
                    Text(match.torrentText ?? match.url)
                        .font(.system(size: 14))
                        .foregroundColor(azureBlue)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .lineLimit(2)
                }
            } else {
                Text(match.torrentText ?? match.url)
                    .font(.system(size: 14))
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .lineLimit(2)
            }

            // Size
            Text(match.fileSize ?? "-")
                .font(.system(size: 13))
                .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))
                .frame(width: 60, alignment: .center)

            // Magnet link
            if let magnetLink = match.magnetLink,
               let magnetURL = Foundation.URL(string: magnetLink) {
                Link(destination: magnetURL) {
                    Image(systemName: "arrow.down.circle")
                        .foregroundColor(azureBlue)
                        .frame(width: 18, height: 18)
                }
                .frame(width: 60, alignment: .center)
            } else {
                Text("-")
                    .font(.system(size: 14))
                    .foregroundColor(Color.gray)
                    .frame(width: 60, alignment: .center)
            }

            // When
            Text(match.created)
                .font(.system(size: 13))
                .frame(width: 100, alignment: .leading)
                .lineLimit(1)

            // Actions
            HStack(spacing: 4) {
                // Soft delete (hide)
                Button(action: onSoftDelete) {
                    Image(systemName: "eye.slash")
                        .foregroundColor(Color.orange)
                        .frame(width: 18, height: 18)
                }
                .buttonStyle(PlainButtonStyle())

                // Hard delete
                Button(action: onHardDelete) {
                    Image(systemName: "trash")
                        .foregroundColor(Color(red: 0.82, green: 0.2, blue: 0.22))
                        .frame(width: 18, height: 18)
                }
                .buttonStyle(PlainButtonStyle())
            }
            .frame(width: 80, alignment: .center)
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
}

#Preview {
    MatchesView(matches: .constant([
        Match(id: 1, item: "Test Item", url: "https://example.com", site: "TestSite",
              torrentText: "Test Torrent", magnetLink: "magnet:?xt=test",
              fileSize: "1.2 GB", created: "2024-01-01 12:00")
    ]))
}
