//
//  URLsView.swift
//  TorrentSeeker
//
//  URLs (Sites) tab view matching the React frontend
//

import SwiftUI

struct URLsView: View {
    @Binding var urls: [SiteURL]
    @State private var newURLText = ""

    let azureBlue = Color(red: 0.0, green: 0.47, blue: 0.831)
    let borderColor = Color(red: 0.882, green: 0.882, blue: 0.882)

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            // Add URL input
            HStack(spacing: 12) {
                TextField("Enter a URL to scrape…", text: $newURLText)
                    .textFieldStyle(RoundedTextFieldStyle())
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .onSubmit {
                        addURL()
                    }

                Button(action: addURL) {
                    Text("Add")
                        .font(.system(size: 14, weight: .medium))
                        .foregroundColor(.white)
                        .padding(.horizontal, 20)
                        .padding(.vertical, 10)
                        .background(azureBlue)
                        .cornerRadius(4)
                }
            }

            // URLs list with card layout
            ForEach(urls) { url in
                URLCard(url: url, onDelete: {
                    deleteURL(id: url.id)
                }, onUpdate: { newURL, newDisplayName in
                    updateURL(id: url.id, url: newURL, displayName: newDisplayName)
                })
            }
        }
    }

    func addURL() {
        let text = newURLText.trimmingCharacters(in: .whitespaces)
        guard !text.isEmpty else { return }

        Task {
            do {
                try await APIService.shared.addURL(url: text)
                newURLText = ""
                urls = try await APIService.shared.getURLs()
            } catch {
                print("Error adding URL: \(error)")
            }
        }
    }

    func updateURL(id: Int, url: String?, displayName: String?) {
        Task {
            do {
                try await APIService.shared.updateURL(id: id, url: url, displayName: displayName)
                urls = try await APIService.shared.getURLs()
            } catch {
                print("Error updating URL: \(error)")
            }
        }
    }

    func deleteURL(id: Int) {
        Task {
            do {
                try await APIService.shared.deleteURL(id: id)
                urls = try await APIService.shared.getURLs()
            } catch {
                print("Error deleting URL: \(error)")
            }
        }
    }
}

struct URLCard: View {
    let url: SiteURL
    let onDelete: () -> Void
    let onUpdate: (String?, String?) -> Void

    @State private var editDisplayName: String
    @State private var editURL: String
    @FocusState private var displayNameFocused: Bool
    @FocusState private var urlFocused: Bool

    init(url: SiteURL, onDelete: @escaping () -> Void, onUpdate: @escaping (String?, String?) -> Void) {
        self.url = url
        self.onDelete = onDelete
        self.onUpdate = onUpdate
        _editDisplayName = State(initialValue: url.displayName ?? "")
        _editURL = State(initialValue: url.url)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            // Actions row at top
            HStack {
                Spacer()

                // Delete button
                Button(action: onDelete) {
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

            // Display Name
            VStack(alignment: .leading, spacing: 2) {
                Text("DISPLAY NAME")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))

                TextField("Display name...", text: $editDisplayName)
                    .textFieldStyle(PlainTextFieldStyle())
                    .font(.system(size: 13))
                    .padding(8)
                    .background(Color(red: 0.98, green: 0.98, blue: 0.98))
                    .cornerRadius(4)
                    .overlay(
                        RoundedRectangle(cornerRadius: 4)
                            .stroke(Color(red: 0.882, green: 0.882, blue: 0.882), lineWidth: 1)
                    )
                    .focused($displayNameFocused)
                    .onSubmit {
                        displayNameFocused = false
                    }
                    .onChange(of: displayNameFocused) { _, focused in
                        if !focused && editDisplayName != (url.displayName ?? "") {
                            onUpdate(nil, editDisplayName)
                        }
                    }
            }

            // URL
            VStack(alignment: .leading, spacing: 2) {
                Text("URL")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))

                TextField("URL...", text: $editURL)
                    .textFieldStyle(PlainTextFieldStyle())
                    .font(.system(size: 13))
                    .padding(8)
                    .background(Color(red: 0.98, green: 0.98, blue: 0.98))
                    .cornerRadius(4)
                    .overlay(
                        RoundedRectangle(cornerRadius: 4)
                            .stroke(Color(red: 0.882, green: 0.882, blue: 0.882), lineWidth: 1)
                    )
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .focused($urlFocused)
                    .onSubmit {
                        urlFocused = false
                    }
                    .onChange(of: urlFocused) { _, focused in
                        if !focused && editURL != url.url {
                            onUpdate(editURL, nil)
                        }
                    }
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
    URLsView(urls: .constant([
        SiteURL(id: 1, url: "https://example.com", displayName: "Example Site", config: nil),
        SiteURL(id: 2, url: "https://test.com", displayName: nil, config: nil)
    ]))
}
