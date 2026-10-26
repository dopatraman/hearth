import SwiftUI

/// Small header above each grouped section — a colored dot, a semibold label,
/// and a muted count.
struct SectionHeader: View {
    let label: String
    let dotColor: Color
    let count: Int

    var body: some View {
        HStack(spacing: 8) {
            Circle()
                .fill(dotColor)
                .frame(width: 8, height: 8)
            Text(label)
                .font(.hearthBody(size: 14, weight: .semibold))
                .foregroundStyle(Color.hearthTextPrimary)
            Text("\(count)")
                .font(.hearthBody(size: 14, weight: .medium))
                .foregroundStyle(Color.hearthTextSecondary)
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
