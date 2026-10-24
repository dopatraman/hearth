import SwiftUI

/// Presented immediately after a successful swipe-right-to-claim. Lets the
/// claimant (optionally) commit to a completion date. Dismissing without
/// saving is fine — the claim already happened.
struct ClaimDateSheet: View {
    @Environment(\.dismiss) private var dismiss
    @State private var selectedDate: Date = Date().addingTimeInterval(60 * 60 * 24)

    let choreTitle: String
    /// Called with the chosen date on Save, nil on Skip. Parent dismisses.
    let onSave: (Date?) async -> Void

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                Text("When will you do it?")
                    .font(.title3)
                    .foregroundStyle(Color.hearthTextSecondary)

                Text(choreTitle)
                    .font(.title2.weight(.semibold))
                    .foregroundStyle(Color.hearthTextPrimary)
                    .multilineTextAlignment(.center)

                DatePicker("", selection: $selectedDate, in: Date()...)
                    .datePickerStyle(.graphical)
                    .labelsHidden()
                    .padding(.horizontal)
                    .tint(Color.hearthLavender)

                Spacer(minLength: 0)

                HStack(spacing: 12) {
                    Button("Skip") {
                        Task {
                            await onSave(nil)
                            dismiss()
                        }
                    }
                    .buttonStyle(.bordered)
                    .frame(maxWidth: .infinity)

                    Button("Save") {
                        Task {
                            await onSave(selectedDate)
                            dismiss()
                        }
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(Color.hearthLavender)
                    .frame(maxWidth: .infinity)
                }
                .padding(.horizontal)
                .padding(.bottom, 16)
            }
            .padding(.top, 24)
            .background(Color.hearthBackground)
            .navigationBarTitleDisplayMode(.inline)
        }
        .presentationDetents([.medium, .large])
        .presentationDragIndicator(.visible)
    }
}
