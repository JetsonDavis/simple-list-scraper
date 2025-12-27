//
//  URLsView.swift
//  TorrentSeeker
//
//  URLs (Sites) tab view matching the React frontend
//

import SwiftUI

struct URLsView: View {
    @Binding var urls: [URL]
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

            // URLs table
            VStack(spacing: 0) {
                // Table header
                HStack {
                    Text("Display Name")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))
                        .frame(width: 200, alignment: .leading)
                        .textCase(.uppercase)

                    Text("URL")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .textCase(.uppercase)

                    Text("Actions")
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
                ForEach(urls) { url in
                    URLRow(url: url, onDelete: {
                        deleteURL(id: url.id)
                    }, onUpdate: { newURL, newDisplayName in
                        updateURL(id: url.id, url: newURL, displayName: newDisplayName)
                    })
                }
            }
            .background(Color.white)
            .cornerRadius(4)
            .overlay(
                RoundedRectangle(cornerRadius: 4)
                    .stroke(borderColor, lineWidth: 1)
            )
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

struct URLRow: View {
    let url: URL
    let onDelete: () -> Void
    let onUpdate: (String?, String?) -> Void

    @State private var editDisplayName: String
    @State private var editURL: String
    @FocusState private var displayNameFocused: Bool
    @FocusState private var urlFocused: Bool

    init(url: URL, onDelete: @escaping () -> Void, onUpdate: @escaping (String?, String?) -> Void) {
        self.url = url
        self.onDelete = onDelete
        self.onUpdate = onUpdate
        _editDisplayName = State(initialValue: url.displayName ?? "")
        _editURL = State(initialValue: url.url)
    }

    var body: some View {
        HStack(spacing: 0) {
            // Display name field
            TextField("Display name...", text: $editDisplayName)
                .textFieldStyle(PlainTextFieldStyle())
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .frame(width: 200)
                .focused($displayNameFocused)
                .onSubmit {
                    displayNameFocused = false
                }
                .onChange(of: displayNameFocused) { _, focused in
                    if !focused && editDisplayName != (url.displayName ?? "") {
                        onUpdate(nil, editDisplayName)
                    }
                }

            // URL field
            TextField("", text: $editURL)
                .textFieldStyle(PlainTextFieldStyle())
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .focused($urlFocused)
                .onSubmit {
                    urlFocused = false
                }
                .onChange(of: urlFocused) { _, focused in
                    if !focused && editURL != url.url {
                        onUpdate(editURL, nil)
                    }
                }

            // Delete button
            Button(action: onDelete) {
                Text("Delete")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))
                    .padding(.horizontal, 16)
                    .padding(.vertical, 8)
                    .background(Color.white)
                    .cornerRadius(4)
                    .overlay(
                        RoundedRectangle(cornerRadius: 4)
                            .stroke(Color(red: 0.882, green: 0.882, blue: 0.882), lineWidth: 1)
                    )
            }
            .frame(width: 100)
            .buttonStyle(PlainButtonStyle())
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
    URLsView(urls: .constant([
        URL(id: 1, url: "https://example.com", displayName: "Example Site", config: nil),
        URL(id: 2, url: "https://test.com", displayName: nil, config: nil)
    ]))
}
