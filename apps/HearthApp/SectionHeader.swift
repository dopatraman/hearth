import SwiftUI

/// Small header above each grouped section — a colored dot, a semibold label,
/// and a muted count.
struct SectionHeader: View {
    let label: String
    let dotColor: Color
    let count: Int
    let hint: String?
    let symbolName: String?
    let countText: String?

    init(
        label: String,
        dotColor: Color,
        count: Int,
        hint: String? = nil,
        symbolName: String? = nil,
        countText: String? = nil
    ) {
        self.label = label
        self.dotColor = dotColor
        self.count = count
        self.hint = hint
        self.symbolName = symbolName
        self.countText = countText
    }

    var body: some View {
        HStack(spacing: 8) {
            if let symbolName {
                Image(systemName: symbolName)
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(dotColor)
                    .frame(width: 14, height: 14)
            } else {
                Circle()
                    .fill(dotColor)
                    .frame(width: 8, height: 8)
            }
            Text(label)
                .font(.hearthBody(size: 14, weight: .semibold))
                .foregroundStyle(Color.hearthTextPrimary)
            Text(countText ?? "\(count)")
                .font(.hearthBody(size: 14, weight: .medium))
                .foregroundStyle(Color.hearthTextSecondary)
            if let hint {
                Spacer(minLength: 8)
                Text(hint)
                    .font(.hearthBody(size: 12, weight: .medium))
                    .foregroundStyle(Color.hearthTextSecondary)
            }
        }
        .padding(.horizontal, 6)
        .padding(.bottom, 10)
    }
}

#if DEBUG
#Preview {
    VStack(alignment: .leading, spacing: 16) {
        SectionHeader(label: "Claimed", dotColor: .hearthLavender, count: 3)
        SectionHeader(label: "Up for grabs", dotColor: .hearthSectionDotMuted, count: 2)
    }
    .padding()
    .background(Color.hearthBackground)
}
#endif
