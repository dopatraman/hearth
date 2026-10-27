import SwiftUI
import UIKit

@Observable
@MainActor
final class NewChoreCardModel {
    var title: String = ""
    var note: String = ""
    var claimForSelf: Bool = false
    var selectedBucket: WhenishBucket?

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
        claimForSelf = false
        selectedBucket = nil
    }
}

struct NewChoreCard: View {
    @Binding var isOpen: Bool
    let onSubmit: (String, String?, Bool, WhenishBucket?) async -> Void

    @State private var model = NewChoreCardModel()
    @FocusState private var titleFocused: Bool

    private let shutSize: CGFloat = 62
    private let openWidth: CGFloat = 358
    private let openHeightCollapsed: CGFloat = 326
    private let openHeightExpanded: CGFloat = 420

    private var openHeight: CGFloat {
        model.claimForSelf ? openHeightExpanded : openHeightCollapsed
    }

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

            VStack(alignment: .leading, spacing: 14) {
                claimToggleRow
                if model.claimForSelf {
                    whenishRow
                        .transition(
                            .asymmetric(
                                insertion: .opacity.combined(with: .move(edge: .top)),
                                removal: .opacity.combined(with: .move(edge: .top))
                            )
                        )
                }
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

    private var claimToggleRow: some View {
        Button {
            UIImpactFeedbackGenerator(style: .light).impactOccurred()
            withAnimation(.timingCurve(0.2, 0.8, 0.2, 1, duration: 0.4)) {
                model.claimForSelf.toggle()
                if !model.claimForSelf {
                    model.selectedBucket = nil
                }
            }
        } label: {
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text("I'll take it")
                        .font(.hearthBody(size: 15, weight: .medium))
                        .foregroundStyle(Color.hearthTextPrimary)
                    Text("Claim it as soon as it's added")
                        .font(.hearthBody(size: 13))
                        .foregroundStyle(Color.hearthTextSecondary)
                }
                Spacer()
                ZStack(alignment: model.claimForSelf ? .trailing : .leading) {
                    Capsule()
                        .fill(model.claimForSelf ? Color.hearthRingMineStroke : Color.hearthScheduledTrack)
                        .frame(width: 48, height: 28)
                    Circle()
                        .fill(Color.white)
                        .frame(width: 22, height: 22)
                        .shadow(color: Color.black.opacity(0.2), radius: 1, y: 1)
                        .padding(.horizontal, 3)
                }
            }
            .frame(height: 44)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .accessibilityLabel("I'll take it")
        .accessibilityValue(model.claimForSelf ? "On" : "Off")
    }

    private var whenishRow: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("By when?")
                .font(.hearthBody(size: 13, weight: .semibold))
                .foregroundStyle(Color.hearthTextSecondary)
            HStack(spacing: 8) {
                ForEach(WhenishBucket.allCases) { bucket in
                    whenishChip(bucket)
                }
            }
        }
    }

    private func whenishChip(_ bucket: WhenishBucket) -> some View {
        let selected = model.selectedBucket == bucket
        return Button {
            UIImpactFeedbackGenerator(style: .light).impactOccurred()
            withAnimation(.timingCurve(0.2, 1.6, 0.4, 1, duration: 0.25)) {
                model.selectedBucket = selected ? nil : bucket
            }
        } label: {
            Text(bucket.label)
                .font(.hearthBody(size: 13, weight: .semibold))
                .foregroundStyle(selected ? Color.hearthLavenderInk : Color.hearthTextSecondary)
                .padding(.horizontal, 12)
                .frame(height: 32)
                .background(
                    Capsule().fill(selected ? Color.hearthLavender : Color.hearthBackground)
                )
                .scaleEffect(selected ? 1.04 : 1)
        }
        .buttonStyle(.plain)
        .accessibilityLabel(bucket.label)
        .accessibilityAddTraits(selected ? .isSelected : [])
    }

    private func submit() {
        guard model.canSubmit else { return }
        let title = model.trimmedTitle
        let notes = model.trimmedNote
        let claim = model.claimForSelf
        let bucket = claim ? model.selectedBucket : nil
        UIImpactFeedbackGenerator(style: .light).impactOccurred()
        isOpen = false
        titleFocused = false
        model.reset()
        Task { await onSubmit(title, notes, claim, bucket) }
    }

    private func close() {
        isOpen = false
        titleFocused = false
        model.reset()
    }
}

private extension Color {
    static let hearthScheduledTrack = Color(
        light: Color(red: 0xDC / 255, green: 0xD9 / 255, blue: 0xE5 / 255),
        dark: Color(red: 0x4A / 255, green: 0x47 / 255, blue: 0x54 / 255)
    )

    init(light: Color, dark: Color) {
        self.init(uiColor: UIColor { trait in
            trait.userInterfaceStyle == .dark ? UIColor(dark) : UIColor(light)
        })
    }
}

#if DEBUG
#Preview {
    @Previewable @State var open = false
    return ZStack {
        Color.hearthBackground.ignoresSafeArea()
        NewChoreCard(isOpen: $open, onSubmit: { _, _, _, _ in })
            .padding(.trailing, 16)
            .padding(.bottom, 24)
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .bottomTrailing)
    }
}
#endif
