import Foundation

enum HearthAPIError: Error, LocalizedError {
    case transport(Error)
    case unexpectedStatus(Int, String)
    case decoding(Error)

    var errorDescription: String? {
        switch self {
        case .transport(let err):
            return "Network error: \(err.localizedDescription)"
        case .unexpectedStatus(let code, let message):
            return "Server returned \(code): \(message)"
        case .decoding(let err):
            return "Could not read server response: \(err.localizedDescription)"
        }
    }
}

/// Zero-field request body for endpoints that take no payload.
private struct EmptyRequest: Encodable {}

/// Thin async wrapper over URLSession for talking to the Hearth server.
/// Each call maps 1:1 to an endpoint. No retries, no caching — handlers do what
/// handlers do, callers decide what to do with failures.
struct HearthAPI {
    let baseURL: URL
    let session: URLSession

    init(baseURL: URL = URL(string: "https://api.hearth.prakashvenkat.com")!,
         session: URLSession = .shared) {
        self.baseURL = baseURL
        self.session = session
    }

    // MARK: - Auth

    func signInWithApple(idToken: String, displayName: String?) async throws -> AuthAppleResponse {
        let body = AuthAppleRequest(idToken: idToken, displayName: displayName)
        return try await post("/auth/apple", body: body)
    }

    func me(sessionToken: String) async throws -> HearthUser {
        try await get("/me", bearerToken: sessionToken)
    }

    // MARK: - Chores

    func backlog(sessionToken: String) async throws -> [HearthChore] {
        let resp: BacklogResponse = try await get("/chores", bearerToken: sessionToken)
        return resp.chores
    }

    func createChore(sessionToken: String, title: String, notes: String?) async throws -> HearthChore {
        let body = CreateChoreRequest(title: title, notes: notes)
        return try await post("/chores", body: body, bearerToken: sessionToken)
    }

    // MARK: - Invites

    func createInvite(sessionToken: String) async throws -> InviteDTO {
        try await post("/invites", body: EmptyRequest(), bearerToken: sessionToken)
    }

    func acceptInvite(sessionToken: String, code: String) async throws -> AcceptInviteResponse {
        try await post("/invites/\(code)/accept", body: EmptyRequest(), bearerToken: sessionToken)
    }

    // MARK: - Internals

    private func post<RequestBody: Encodable, ResponseBody: Decodable>(
        _ path: String,
        body: RequestBody,
        bearerToken: String? = nil
    ) async throws -> ResponseBody {
        var req = URLRequest(url: baseURL.appendingPathComponent(path))
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let bearerToken {
            req.setValue("Bearer \(bearerToken)", forHTTPHeaderField: "Authorization")
        }
        req.httpBody = try JSONEncoder().encode(body)
        return try await perform(req)
    }

    private func get<ResponseBody: Decodable>(
        _ path: String,
        bearerToken: String? = nil
    ) async throws -> ResponseBody {
        var req = URLRequest(url: baseURL.appendingPathComponent(path))
        req.httpMethod = "GET"
        if let bearerToken {
            req.setValue("Bearer \(bearerToken)", forHTTPHeaderField: "Authorization")
        }
        return try await perform(req)
    }

    private func perform<ResponseBody: Decodable>(_ req: URLRequest) async throws -> ResponseBody {
        let data: Data
        let response: URLResponse
        do {
            (data, response) = try await session.data(for: req)
        } catch {
            throw HearthAPIError.transport(error)
        }

        guard let http = response as? HTTPURLResponse else {
            throw HearthAPIError.unexpectedStatus(0, "non-HTTP response")
        }

        guard (200..<300).contains(http.statusCode) else {
            let message = (try? JSONDecoder().decode(APIErrorResponse.self, from: data).error)
                ?? String(data: data, encoding: .utf8) ?? ""
            throw HearthAPIError.unexpectedStatus(http.statusCode, message)
        }

        do {
            return try JSONDecoder().decode(ResponseBody.self, from: data)
        } catch {
            throw HearthAPIError.decoding(error)
        }
    }
}
