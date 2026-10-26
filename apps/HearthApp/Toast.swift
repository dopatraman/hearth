import SwiftUI

struct ToastModel: Equatable, Identifiable {
    let id = UUID()
    let text: String
    let undoLabel: String?

    static func == (lhs: ToastModel, rhs: ToastModel) -> Bool {
        lhs.id == rhs.id && lhs.text == rhs.text && lhs.undoLabel == rhs.undoLabel
    }
}

/// Dark glass pill toast that rises from above the tab bar. Carries a text
/// label and an optional Undo button. Auto-dismisses after 3.2s via the
/// `.task` modifier bound to the toast's id.
struct ToastView: View {
    let toast: ToastModel
    let onUndo: (() -> Void)?

    var body: some View {
        HStack(spacing: 10) {
            Image(systemName: "checkmark.circle.fill")
                .font(.system(size: 20, weight: .semibold))
                .foregroundStyle(Color.hearthMint)

            Text(toast.text)
                .font(.hearthBody(size: 14, weight: .medium))
                .foregroundStyle(.white)

            if let undoLabel = toast.undoLabel, let onUndo {
                Button(action: onUndo) {
                    Text(undoLabel)
                        .font(.hearthBody(size: 13, weight: .semibold))
                        .foregroundStyle(Color.hearthLavender)
                        .padding(.horizontal, 12)
                        .frame(height: 30)
                        .background(Color.white.opacity(0.12), in: Capsule())
                }
                .buttonStyle(.plain)
            }
        }
        .padding(.leading, 14)
        .padding(.trailing, toast.undoLabel == nil ? 14 : 10)
        .padding(.vertical, 10)
        .background(
            Capsule().fill(.black.opacity(0.9))
        )
        .shadow(color: .black.opacity(0.25), radius: 20, y: 10)
    }
}

#if DEBUG
#Preview {
    VStack(spacing: 20) {
        ToastView(toast: ToastModel(text: "Moved to Done", undoLabel: "Undo"), onUndo: {})
        ToastView(toast: ToastModel(text: "Yours. Due tomorrow?", undoLabel: nil), onUndo: nil)
    }
    .padding()
    .background(Color.hearthBackground)
}
#endif
