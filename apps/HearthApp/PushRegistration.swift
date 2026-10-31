import Foundation
import UIKit
import UserNotifications

@MainActor
final class PushRegistration: NSObject, UNUserNotificationCenterDelegate {
    static let shared = PushRegistration()

    private var pendingToken: String?
    private var sessionProvider: (() -> String?)?
    var onRefresh: (() async -> Void)?
    private let api: HearthAPI

    private override init() {
        self.api = HearthAPI()
        super.init()
    }

    func configure(sessionProvider: @escaping () -> String?) {
        self.sessionProvider = sessionProvider
        UNUserNotificationCenter.current().delegate = self
    }

    func requestPermissionAndRegister() async {
        let center = UNUserNotificationCenter.current()
        do {
            let granted = try await center.requestAuthorization(options: [.alert, .sound, .badge])
            if granted {
                UIApplication.shared.registerForRemoteNotifications()
            }
        } catch {
            // Permission request errors (e.g. provisioning not configured
            // locally) are fine to swallow — the toggle in Settings is the
            // user's path back.
        }
    }

    func didReceiveDeviceToken(_ data: Data) {
        let token = data.map { String(format: "%02x", $0) }.joined()
        pendingToken = token
        Task { await sendPendingTokenIfPossible() }
    }

    func refresh() async {
        await onRefresh?()
    }

    func didFailToRegister(_ error: Error) {
        // No-op for now; token will remain nil, server simply won't push.
    }

    func sendPendingTokenIfPossible() async {
        guard let token = pendingToken,
              let session = sessionProvider?()
        else { return }
        do {
            try await api.registerDevice(sessionToken: session, token: token, platform: "ios")
            pendingToken = nil
        } catch {
            // Keep pendingToken so a later sign-in flush retries.
        }
    }

    nonisolated func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification,
        withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void
    ) {
        Task { @MainActor in await PushRegistration.shared.refresh() }
        completionHandler([.banner, .sound])
    }

    nonisolated func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse,
        withCompletionHandler completionHandler: @escaping () -> Void
    ) {
        Task { @MainActor in await PushRegistration.shared.refresh() }
        completionHandler()
    }
}
