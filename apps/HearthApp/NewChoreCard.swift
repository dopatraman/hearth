import SwiftUI
import UIKit

@Observable
@MainActor
final class NewChoreCardModel {
    var title: String = ""
    var note: String = ""

    var trimmedTitle: String {
        title.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    var trimmedNote: String? {
        let trimmed = note.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }

    var canSubmit: Bool {
        !trimmedTitle.isEmpty
    }

    func reset() {
        title = ""
        note = ""
    }
}

struct NewChoreCard: View {
    @Binding var isOpen: Bool
    let onSubmit: (String, String?) async -> Void

    @State private var model = NewChoreCardModel()
    @FocusState private var titleFocused: Bool

    private let shutSize: CGFloat = 62
    private let openWidth: CGFloat = 358
    private let openHeight: CGFloat = 280

    var body: some View {
        ZStack(alignment: .bottomTrailing) {
            formContent
                .frame(width: openWidth, height: openHeight, alignment: .topLeading)
                .opacity(isOpen ? 1 : 0)
                .allowsHitTesting(isOpen)

            plusContent
                .opacity(isOpen ? 0 : 1)
                .allowsHitTesting(!isOpen)
        }
        .frame(
            width: isOpen ? openWidth : shutSize,
            height: isOpen ? openHeight : shutSize,
            alignment: .bottomTrailing
        )
        .background(
            RoundedRectangle(cornerRadius: isOpen ? 30 : 31, style: .continuous)
                .fill(isOpen ? Color.white : Color.hearthLavender)
        )
        .clipShape(RoundedRectangle(cornerRadius: isOpen ? 30 : 31, style: .continuous))
        .shadow(
            color: isOpen ? Color.black.opacity(0.22) : Color.hearthLavender.opacity(0.38),
            radius: isOpen ? 30 : 20,
            y: isOpen ? 24 : 8
        )
        .animation(.timingCurve(0.32, 1.18, 0.42, 1, duration: 0.55), value: isOpen)
    }

    private var plusContent: some View {
        Button {
            UIImpactFeedbackGenerator(style: .light).impactOccurred()
            isOpen = true
            titleFocused = true
        } label: {
            Image(systemName: "plus")
                .font(.system(size: 24, weight: .semibold))
                .foregroundStyle(Color.hearthLavenderInk)
                .frame(width: shutSize, height: shutSize)
                .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .accessibilityLabel("New chore")
    }

    private var formContent: some View {
        VStack(alignment: .leading, spacing: 20) {
            HStack {
                Text("New chore")
                    .font(.hearthBody(size: 13, weight: .semibold))
                    .foregroundStyle(Color.hearthTextSecondary)
                Spacer()
                Button(action: close) {
                    Image(systemName: "xmark")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(Color.hearthTextSecondary)
                        .frame(width: 36, height: 36)
                        .background(Color.hearthBackground, in: Circle())
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Close")
            }

            VStack(alignment: .leading, spacing: 10) {
                TextField("What needs doing?", text: $model.title, axis: .horizontal)
                    .font(.hearthDisplay(size: 26, weight: .semibold))
                    .foregroundStyle(Color.hearthTextPrimary)
                    .focused($titleFocused)
                    .submitLabel(.next)
                TextField("Add a note, if it helps", text: $model.note)
                    .font(.hearthBody(size: 15))
                    .foregroundStyle(Color.hearthTextSecondary)
                    .submitLabel(.done)
                    .onSubmit(submit)
            }

            Button(action: submit) {
                Text("Add to backlog")
                    .font(.hearthBody(size: 16, weight: .semibold))
                    .foregroundStyle(.white)
                    .frame(maxWidth: .infinity)
                    .frame(height: 54)
                    .background(Color.hearthTextPrimary, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
                    .opacity(model.canSubmit ? 1 : 0.35)
            }
            .buttonStyle(.plain)
            .disabled(!model.canSubmit)
        }
        .padding(22)
    }

    private func submit() {
        guard model.canSubmit else { return }
        let title = model.trimmedTitle
        let notes = model.trimmedNote
        UIImpactFeedbackGenerator(style: .light).impactOccurred()
        isOpen = false
        titleFocused = false
        model.reset()
        Task { await onSubmit(title, notes) }
    }

    private func close() {
        isOpen = false
        titleFocused = false
        model.reset()
    }
}

#if DEBUG
#Preview {
    @Previewable @State var open = false
    return ZStack {
        Color.hearthBackground.ignoresSafeArea()
        NewChoreCard(isOpen: $open, onSubmit: { _, _ in })
            .padding(.trailing, 16)
            .padding(.bottom, 24)
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .bottomTrailing)
    }
}
#endif
