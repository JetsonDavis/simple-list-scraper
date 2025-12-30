//
//  MatchesView.swift
//  TorrentSeeker
//
//  Matches tab view with mobile-friendly card layout
//

import SwiftUI

struct MatchesView: View {
    @Binding var matches: [Match]

    let borderColor = Color(red: 0.882, green: 0.882, blue: 0.882)

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            // Matches list with card layout
            ForEach(matches) { match in
                MatchCard(match: match, onSoftDelete: {
                    softDeleteMatch(id: match.id)
                }, onHardDelete: {
                    hardDeleteMatch(id: match.id)
                })
            }
        }
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

struct MatchCard: View {
    let match: Match
    let onSoftDelete: () -> Void
    let onHardDelete: () -> Void

    let azureBlue = Color(red: 0.0, green: 0.47, blue: 0.831)

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            // Actions row at top
            HStack {
                Spacer()

                // Soft delete (hide)
                Button(action: onSoftDelete) {
                    HStack(spacing: 4) {
                        Image(systemName: "eye.slash")
                        Text("Hide")
                            .font(.system(size: 12))
                    }
                    .foregroundColor(.white)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 6)
                    .background(Color.orange)
                    .cornerRadius(4)
                }

                // Hard delete
                Button(action: onHardDelete) {
                    HStack(spacing: 4) {
                        Image(systemName: "trash")
                        Text("Delete")
                            .font(.system(size: 12))
                    }
                    .foregroundColor(.white)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 6)
                    .background(Color(red: 0.82, green: 0.2, blue: 0.22))
                    .cornerRadius(4)
                }
            }

            // Item
            VStack(alignment: .leading, spacing: 2) {
                Text("ITEM")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))

                Text(match.item)
                    .font(.system(size: 13))
                    .foregroundColor(.primary)
            }

            // Site
            VStack(alignment: .leading, spacing: 2) {
                Text("SITE")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))

                Text(match.site)
                    .font(.system(size: 13))
                    .foregroundColor(.primary)
            }

            // Torrent text with link
            VStack(alignment: .leading, spacing: 2) {
                Text("TORRENT TEXT")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))

                if let urlString = Foundation.URL(string: match.url) {
                    Link(destination: urlString) {
                        Text(match.torrentText ?? match.url)
                            .font(.system(size: 13))
                            .foregroundColor(azureBlue)
                            .underline()
                            .fixedSize(horizontal: false, vertical: true)
                    }
                } else {
                    Text(match.torrentText ?? match.url)
                        .font(.system(size: 13))
                        .foregroundColor(.primary)
                        .fixedSize(horizontal: false, vertical: true)
                }
            }

            // Size and Magnet in a row
            HStack(spacing: 20) {
                // Size
                VStack(alignment: .leading, spacing: 2) {
                    Text("SIZE")
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))

                    Text(match.fileSize ?? "-")
                        .font(.system(size: 13))
                        .foregroundColor(.primary)
                }

                // Magnet link
                VStack(alignment: .leading, spacing: 2) {
                    Text("MAGNET")
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))

                    if let magnetLink = match.magnetLink,
                       let magnetURL = Foundation.URL(string: magnetLink) {
                        Link(destination: magnetURL) {
                            HStack(spacing: 4) {
                                Image(systemName: "arrow.down.circle")
                                Text("Download")
                                    .font(.system(size: 12))
                            }
                            .foregroundColor(azureBlue)
                        }
                    } else {
                        Text("-")
                            .font(.system(size: 13))
                            .foregroundColor(Color.gray)
                    }
                }
            }

            // When
            VStack(alignment: .leading, spacing: 2) {
                Text("WHEN")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))

                Text(match.created)
                    .font(.system(size: 13))
                    .foregroundColor(.primary)
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
}

#Preview {
    MatchesView(matches: .constant([
        Match(id: 1, item: "Test Item", url: "https://example.com", site: "TestSite",
              torrentText: "Test Torrent", magnetLink: "magnet:?xt=test",
              fileSize: "1.2 GB", created: "2024-01-01 12:00")
    ]))
}
