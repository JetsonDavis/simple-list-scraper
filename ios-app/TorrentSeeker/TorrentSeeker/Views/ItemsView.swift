//
//  ItemsView.swift
//  TorrentSeeker
//
//  Items tab view matching the React frontend
//

import SwiftUI

struct ItemsView: View {
    @Binding var items: [Item]
    @State private var newItemText = ""

    let azureBlue = Color(red: 0.0, green: 0.47, blue: 0.831)
    let borderColor = Color(red: 0.882, green: 0.882, blue: 0.882)

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            // Add item input
            HStack(spacing: 12) {
                TextField("Type an item…", text: $newItemText)
                    .textFieldStyle(RoundedTextFieldStyle())
                    .onSubmit {
                        addItem()
                    }

                Button(action: addItem) {
                    Text("Add")
                        .font(.system(size: 14, weight: .medium))
                        .foregroundColor(.white)
                        .padding(.horizontal, 20)
                        .padding(.vertical, 10)
                        .background(azureBlue)
                        .cornerRadius(4)
                }
            }

            // Items table
            VStack(spacing: 0) {
                // Table header
                HStack {
                    Text("Actions")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))
                        .frame(width: 100)
                        .textCase(.uppercase)

                    Text("Item")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundColor(Color(red: 0.376, green: 0.369, blue: 0.361))
                        .frame(maxWidth: .infinity, alignment: .leading)
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
                ForEach(items) { item in
                    ItemRow(item: item, onDelete: {
                        deleteItem(id: item.id)
                    }, onUpdate: { newText in
                        updateItem(id: item.id, text: newText)
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

    func addItem() {
        let text = newItemText.trimmingCharacters(in: .whitespaces)
        guard !text.isEmpty else { return }

        Task {
            do {
                try await APIService.shared.addItem(text: text)
                newItemText = ""
                items = try await APIService.shared.getItems()
            } catch {
                print("Error adding item: \(error)")
            }
        }
    }

    func updateItem(id: Int, text: String) {
        let text = text.trimmingCharacters(in: .whitespaces)
        guard !text.isEmpty else { return }

        Task {
            do {
                try await APIService.shared.updateItem(id: id, text: text)
                items = try await APIService.shared.getItems()
            } catch {
                print("Error updating item: \(error)")
            }
        }
    }

    func deleteItem(id: Int) {
        Task {
            do {
                try await APIService.shared.deleteItem(id: id)
                items = try await APIService.shared.getItems()
            } catch {
                print("Error deleting item: \(error)")
            }
        }
    }
}

struct ItemRow: View {
    let item: Item
    let onDelete: () -> Void
    let onUpdate: (String) -> Void

    @State private var editText: String
    @FocusState private var isFocused: Bool

    init(item: Item, onDelete: @escaping () -> Void, onUpdate: @escaping (String) -> Void) {
        self.item = item
        self.onDelete = onDelete
        self.onUpdate = onUpdate
        _editText = State(initialValue: item.text)
    }

    var body: some View {
        HStack(spacing: 0) {
            // Delete button
            Button(action: onDelete) {
                Image(systemName: "trash")
                    .foregroundColor(Color(red: 0.82, green: 0.2, blue: 0.22))
                    .frame(width: 18, height: 18)
            }
            .frame(width: 100)
            .buttonStyle(PlainButtonStyle())

            // Editable text field
            TextField("", text: $editText)
                .textFieldStyle(PlainTextFieldStyle())
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .focused($isFocused)
                .onSubmit {
                    isFocused = false
                }
                .onChange(of: isFocused) { _, focused in
                    if !focused && editText != item.text {
                        onUpdate(editText)
                    }
                }
        }
        .padding(.vertical, 8)
        .background(Color.white)
        .overlay(
            Rectangle()
                .frame(height: 1)
                .foregroundColor(Color(red: 0.953, green: 0.949, blue: 0.945)),
            alignment: .bottom
        )
    }
}

struct RoundedTextFieldStyle: TextFieldStyle {
    func _body(configuration: TextField<Self._Label>) -> some View {
        configuration
            .padding(.horizontal, 14)
            .padding(.vertical, 10)
            .background(Color.white)
            .cornerRadius(4)
            .overlay(
                RoundedRectangle(cornerRadius: 4)
                    .stroke(Color(red: 0.882, green: 0.882, blue: 0.882), lineWidth: 1)
            )
    }
}

#Preview {
    ItemsView(items: .constant([
        Item(id: 1, text: "Test item 1"),
        Item(id: 2, text: "Test item 2")
    ]))
}
