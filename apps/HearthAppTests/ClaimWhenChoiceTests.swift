import Foundation
import Testing
@testable import HearthApp

@Suite("ClaimWhenChoice")
struct ClaimWhenChoiceTests {

    @Test("non-custom choices map to a WhenishBucket")
    func bucket_mapsForDateChoices() {
        #expect(ClaimWhenChoice.tonight.bucket == .tonight)
        #expect(ClaimWhenChoice.tomorrow.bucket == .tomorrow)
        #expect(ClaimWhenChoice.weekend.bucket == .weekend)
    }

    @Test("custom has no bucket (its date comes from the inline picker)")
    func bucket_nilForCustom() {
        #expect(ClaimWhenChoice.custom.bucket == nil)
    }

    @Test("labels match the user-facing copy")
    func labels_match() {
        #expect(ClaimWhenChoice.tonight.label == "Tonight")
        #expect(ClaimWhenChoice.tomorrow.label == "Tomorrow")
        #expect(ClaimWhenChoice.weekend.label == "This weekend")
        #expect(ClaimWhenChoice.custom.label == "Custom")
    }

    @Test("all four choices surface via allCases, in order")
    func allCases_order() {
        #expect(ClaimWhenChoice.allCases == [.tonight, .tomorrow, .weekend, .custom])
    }
}
