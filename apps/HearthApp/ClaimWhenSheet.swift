import SwiftUI
import UIKit

/// Choice offered on the Claim-when sheet after a user claims an unclaimed
/// chore (or taps "Set a date" on their own claim). Mirrors the New Chore
/// composer's When-ish chips, plus Custom. Tonight/Tomorrow/Weekend commit
/// immediately with a resolved date; Custom reveals a graphical picker in
/// the same sheet.
enum ClaimWhenChoice: CaseIterable, Hashable {
    case tonight, tomorrow, weekend, custom

    var label: String {
        switch self {
        case .tonight: return "Tonight"
        case .tomorrow: return "Tomorrow"
        case .weekend: return "This weekend"
        case .custom: return "Custom"
        }
    }

    var bucket: WhenishBucket? {
        switch self {
        case .tonight: return .tonight
        case .tomorrow: return .tomorrow
        case .weekend: return .weekend
        case .custom: return nil
        }
    }
}

struct ClaimWhenSheet: View {
    @Environment(\.dismiss) private var dismiss
    @State private var isCustomMode = false
    @State private var customDate: Date = Date().addingTimeInterval(24 * 60 * 60)
    @State private var detent: PresentationDetent = .medium

    let choreTitle: String
    /// Called with the chosen Date on commit; nil on X-dismiss. Parent
    /// usually guards `if let date` and skips the API write on nil.
    let onSave: (Date?) async -> Void

    var body: some View {
        NavigationStack {
            VStack(alignment: .leading, spacing: 20) {
                header

                WrappingHStack(spacing: 8, rowSpacing: 8) {
                    ForEach(ClaimWhenChoice.allCases, id: \.self) { choice in
                        chip(choice)
                    }
                }

                if isCustomMode {
                    customPicker
                        .transition(
                            .asymmetric(
                                insertion: .opacity.combined(with: .move(edge: .top)),
                                removal: .opacity.combined(with: .move(edge: .top))
                            )
                        )
                }

                Spacer(minLength: 0)
            }
            .padding(20)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(Color.hearthBackground)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        dismiss()
                    } label: {
                        Image(systemName: "xmark")
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(Color.hearthTextSecondary)
                            .frame(width: 32, height: 32)
                            .background(Color.hearthInsetFill, in: Circle())
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel("Close")
                }
            }
        }
        .presentationDetents([.medium, .large], selection: $detent)
        .presentationDragIndicator(.visible)
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text("When will you do it?")
                .font(.hearthBody(size: 15, weight: .medium))
                .foregroundStyle(Color.hearthTextSecondary)
            Text(choreTitle)
                .font(.hearthDisplay(size: 24, weight: .semibold))
                .foregroundStyle(Color.hearthTextPrimary)
        }
    }

    @ViewBuilder
    private func chip(_ choice: ClaimWhenChoice) -> some View {
        let selected = choice == .custom && isCustomMode
        Button {
            tap(choice)
        } label: {
            Text(choice.label)
                .font(.hearthBody(size: 14, weight: .semibold))
                .foregroundStyle(selected ? Color.hearthLavenderInk : Color.hearthTextSecondary)
                .fixedSize(horizontal: true, vertical: false)
                .padding(.horizontal, 14)
                .frame(height: 36)
                .background(
                    Capsule().fill(selected ? Color.hearthLavender : Color.hearthInsetFill)
                )
                .scaleEffect(selected ? 1.04 : 1)
        }
        .buttonStyle(.plain)
        .accessibilityLabel(choice.label)
    }

    private var customPicker: some View {
        VStack(alignment: .leading, spacing: 16) {
            DatePicker(
                "",
                selection: $customDate,
                in: Date()...,
                displayedComponents: .date
            )
            .datePickerStyle(.graphical)
            .labelsHidden()
            .tint(Color.hearthLavender)

            Button(action: saveCustom) {
                Text("Save")
                    .font(.hearthBody(size: 16, weight: .semibold))
                    .foregroundStyle(.white)
                    .frame(maxWidth: .infinity)
                    .frame(height: 54)
                    .background(
                        Color.hearthTextPrimary,
                        in: RoundedRectangle(cornerRadius: 18, style: .continuous)
                    )
            }
            .buttonStyle(.plain)
        }
    }

    private func tap(_ choice: ClaimWhenChoice) {
        UIImpactFeedbackGenerator(style: .light).impactOccurred()
        if choice == .custom {
            withAnimation(.timingCurve(0.2, 0.8, 0.2, 1, duration: 0.4)) {
                isCustomMode.toggle()
            }
            detent = isCustomMode ? .large : .medium
            return
        }
        guard let bucket = choice.bucket,
              let date = bucket.resolve(now: Date()) else { return }
        Task {
            await onSave(date)
            dismiss()
        }
    }

    private func saveCustom() {
        let endOfDay = Calendar.current
            .startOfDay(for: customDate)
            .addingTimeInterval(24 * 60 * 60 - 1)
        Task {
            await onSave(endOfDay)
            dismiss()
        }
    }
}

#if DEBUG
#Preview {
    Color.hearthBackground
        .sheet(isPresented: .constant(true)) {
            ClaimWhenSheet(choreTitle: "Clean out the fridge") { _ in }
        }
}
#endif
