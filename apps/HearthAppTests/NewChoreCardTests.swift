import Foundation
import Testing
@testable import HearthApp

@MainActor
@Suite("NewChoreCard model")
struct NewChoreCardTests {

    @Test("canSubmit is false when title is empty")
    func canSubmit_falseWhenEmpty() {
        let model = NewChoreCardModel()
        #expect(model.canSubmit == false)
    }

    @Test("canSubmit is false when title is whitespace only")
    func canSubmit_falseWhenWhitespace() {
        let model = NewChoreCardModel()
        model.title = "   \n\t  "
        #expect(model.canSubmit == false)
    }

    @Test("canSubmit is true when title has content")
    func canSubmit_trueWhenContent() {
        let model = NewChoreCardModel()
        model.title = "Descale the kettle"
        #expect(model.canSubmit == true)
    }

    @Test("trimmedTitle strips surrounding whitespace")
    func trimmedTitle_stripsWhitespace() {
        let model = NewChoreCardModel()
        model.title = "  Mop the kitchen  "
        #expect(model.trimmedTitle == "Mop the kitchen")
    }

    @Test("trimmedNote returns nil for empty or whitespace")
    func trimmedNote_nilForEmpty() {
        let model = NewChoreCardModel()
        model.note = ""
        #expect(model.trimmedNote == nil)
        model.note = "   "
        #expect(model.trimmedNote == nil)
    }

    @Test("trimmedNote returns trimmed content when present")
    func trimmedNote_trimmedWhenPresent() {
        let model = NewChoreCardModel()
        model.note = "  front porch  "
        #expect(model.trimmedNote == "front porch")
    }

    @Test("reset clears title, note, and claim toggle")
    func reset_clearsFields() {
        let model = NewChoreCardModel()
        model.title = "x"
        model.note = "y"
        model.claimForSelf = true
        model.reset()
        #expect(model.title == "")
        #expect(model.note == "")
        #expect(model.claimForSelf == false)
        #expect(model.canSubmit == false)
    }

    @Test("claimForSelf defaults to false")
    func claimForSelf_defaultsFalse() {
        let model = NewChoreCardModel()
        #expect(model.claimForSelf == false)
    }

    @Test("selectedBucket defaults to nil")
    func selectedBucket_defaultsNil() {
        let model = NewChoreCardModel()
        #expect(model.selectedBucket == nil)
    }

    @Test("reset clears selectedBucket")
    func reset_clearsBucket() {
        let model = NewChoreCardModel()
        model.selectedBucket = .weekend
        model.reset()
        #expect(model.selectedBucket == nil)
    }
}
