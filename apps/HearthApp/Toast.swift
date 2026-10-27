import SwiftUI

enum ToastKind: Equatable {
    case success
    case creation
}

struct ToastModel: Equatable, Identifiable {
    let id = UUID()
    let text: String
    let undoLabel: String?
    let kind: ToastKind

    init(text: String, undoLabel: String? = nil, kind: ToastKind = .success) {
        self.text = text
        self.undoLabel = undoLabel
        self.kind = kind
    }

    static func == (lhs: ToastModel, rhs: ToastModel) -> Bool {
        lhs.id == rhs.id && lhs.text == rhs.text && lhs.undoLabel == rhs.undoLabel && lhs.kind == rhs.kind
    }
}

struct ToastView: View {
    let toast: ToastModel
    let onUndo: (() -> Void)?

    var body: some View {
        HStack(spacing: 10) {
            leadingIcon

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

    @ViewBuilder
    private var leadingIcon: some View {
        switch toast.kind {
        case .success:
            Image(systemName: "checkmark.circle.fill")
                .font(.system(size: 20, weight: .semibold))
                .foregroundStyle(Color.hearthMint)
        case .creation:
            ZStack {
                Circle().fill(Color.hearthLavender)
                Image(systemName: "plus")
                    .font(.system(size: 11, weight: .bold))
                    .foregroundStyle(Color.hearthLavenderInk)
            }
            .frame(width: 22, height: 22)
        }
    }
}

#if DEBUG
#Preview {
    VStack(spacing: 20) {
        ToastView(toast: ToastModel(text: "Moved to Done", undoLabel: "Undo"), onUndo: {})
        ToastView(toast: ToastModel(text: "Yours. Due tomorrow?"), onUndo: nil)
        ToastView(toast: ToastModel(text: "Added. Up for grabs", kind: .creation), onUndo: nil)
    }
    .padding()
    .background(Color.hearthBackground)
}
#endif
