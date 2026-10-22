import SwiftUI

struct CreateItemSheet: View {
    @Environment(\.dismiss) private var dismiss
    @State private var title: String = ""
    @State private var notes: String = ""
    @FocusState private var titleFocused: Bool

    /// Called when the user taps Save. Receives (title, notes-or-nil-if-empty).
    /// The parent sheet dismisses after this returns.
    let onSave: (String, String?) async -> Void

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Title", text: $title)
                        .focused($titleFocused)
                        .submitLabel(.done)
                        .onSubmit(trySave)
                }
                Section("Notes") {
                    TextField("Optional", text: $notes, axis: .vertical)
                        .lineLimit(3...6)
                }
            }
            .navigationTitle("New chore")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") { trySave() }
                        .disabled(trimmedTitle.isEmpty)
                        .fontWeight(.semibold)
                }
            }
            .onAppear { titleFocused = true }
            .scrollContentBackground(.hidden)
            .background(Color.hearthBackground)
        }
        .presentationDetents([.medium, .large])
        .presentationDragIndicator(.visible)
    }

    private var trimmedTitle: String {
        title.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private func trySave() {
        let t = trimmedTitle
        guard !t.isEmpty else { return }
        let n: String? = {
            let trimmed = notes.trimmingCharacters(in: .whitespacesAndNewlines)
            return trimmed.isEmpty ? nil : trimmed
        }()
        Task {
            await onSave(t, n)
            dismiss()
        }
    }
}
