//
//  LoginView.swift
//  TorrentSeeker
//
//  Login screen matching the React frontend design
//

import SwiftUI

struct LoginView: View {
    @EnvironmentObject var authViewModel: AuthViewModel

    @State private var isLogin = true
    @State private var username = ""
    @State private var password = ""
    @State private var error = ""
    @State private var loading = false

    var body: some View {
        ZStack {
            // Gradient background matching CSS
            LinearGradient(
                gradient: Gradient(colors: [
                    Color(red: 0.4, green: 0.495, blue: 0.918),  // #667eea
                    Color(red: 0.463, green: 0.294, blue: 0.635) // #764ba2
                ]),
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
            .ignoresSafeArea()

            VStack(spacing: 0) {
                // White card container
                VStack(spacing: 0) {
                    // Title
                    Text("Torrent Seeker")
                        .font(.system(size: 28, weight: .semibold))
                        .foregroundColor(Color(red: 0.2, green: 0.2, blue: 0.2))
                        .padding(.top, 40)
                        .padding(.bottom, 30)

                    // Login/Register toggle
                    HStack(spacing: 0) {
                        Button(action: { isLogin = true }) {
                            Text("Login")
                                .font(.system(size: 16, weight: .medium))
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 12)
                                .background(isLogin ? Color(red: 0.4, green: 0.495, blue: 0.918) : Color.white)
                                .foregroundColor(isLogin ? .white : Color(red: 0.4, green: 0.4, blue: 0.4))
                        }

                        Button(action: { isLogin = false }) {
                            Text("Register")
                                .font(.system(size: 16, weight: .medium))
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 12)
                                .background(!isLogin ? Color(red: 0.4, green: 0.495, blue: 0.918) : Color.white)
                                .foregroundColor(!isLogin ? .white : Color(red: 0.4, green: 0.4, blue: 0.4))
                        }
                    }
                    .background(Color.white)
                    .cornerRadius(8)
                    .overlay(
                        RoundedRectangle(cornerRadius: 8)
                            .stroke(Color(red: 0.88, green: 0.88, blue: 0.88), lineWidth: 1)
                    )
                    .padding(.horizontal, 40)
                    .padding(.bottom, 30)

                    // Form
                    VStack(alignment: .leading, spacing: 20) {
                        // Username field
                        VStack(alignment: .leading, spacing: 8) {
                            Text("Username")
                                .font(.system(size: 14, weight: .medium))
                                .foregroundColor(Color(red: 0.33, green: 0.33, blue: 0.33))

                            TextField("Enter username", text: $username)
                                .textFieldStyle(CustomTextFieldStyle())
                                .textInputAutocapitalization(.never)
                                .autocorrectionDisabled()
                        }

                        // Password field
                        VStack(alignment: .leading, spacing: 8) {
                            Text("Password")
                                .font(.system(size: 14, weight: .medium))
                                .foregroundColor(Color(red: 0.33, green: 0.33, blue: 0.33))

                            SecureField("Enter password", text: $password)
                                .textFieldStyle(CustomTextFieldStyle())
                        }

                        // Error message
                        if !error.isEmpty {
                            Text(error)
                                .font(.system(size: 14))
                                .foregroundColor(Color(red: 0.8, green: 0.2, blue: 0.2))
                                .padding(12)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .background(Color(red: 1.0, green: 0.93, blue: 0.93))
                                .cornerRadius(6)
                        }

                        // Submit button
                        Button(action: handleSubmit) {
                            Text(loading ? "Please wait..." : (isLogin ? "Login" : "Register"))
                                .font(.system(size: 16, weight: .semibold))
                                .foregroundColor(.white)
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 14)
                                .background(loading ? Color.gray : Color(red: 0.4, green: 0.495, blue: 0.918))
                                .cornerRadius(6)
                        }
                        .disabled(loading)
                    }
                    .padding(.horizontal, 40)

                    // Footer link
                    HStack {
                        Text(isLogin ? "New user? " : "Already have an account? ")
                            .font(.system(size: 13))
                            .foregroundColor(Color(red: 0.53, green: 0.53, blue: 0.53))

                        Button(action: {
                            isLogin.toggle()
                            error = ""
                        }) {
                            Text(isLogin ? "Register here" : "Login here")
                                .font(.system(size: 13, weight: .medium))
                                .foregroundColor(Color(red: 0.4, green: 0.495, blue: 0.918))
                        }
                    }
                    .padding(.top, 20)
                    .padding(.bottom, 40)
                }
                .background(Color.white)
                .cornerRadius(12)
                .shadow(color: Color.black.opacity(0.2), radius: 20, x: 0, y: 10)
                .padding(.horizontal, 32)
            }
        }
    }

    func handleSubmit() {
        guard !username.isEmpty && !password.isEmpty else {
            error = "Please fill in all fields"
            return
        }

        guard username.count >= 3 else {
            error = "Username must be at least 3 characters"
            return
        }

        guard password.count >= 6 else {
            error = "Password must be at least 6 characters"
            return
        }

        error = ""
        loading = true

        Task {
            do {
                let response: AuthResponse
                if isLogin {
                    response = try await APIService.shared.login(username: username, password: password)
                } else {
                    response = try await APIService.shared.register(username: username, password: password)
                }

                await MainActor.run {
                    authViewModel.login(token: response.token, username: response.username)
                    loading = false
                }
            } catch APIError.serverError(let message) {
                await MainActor.run {
                    self.error = message
                    loading = false
                }
            } catch {
                await MainActor.run {
                    self.error = "Network error. Please try again."
                    loading = false
                }
            }
        }
    }
}

struct CustomTextFieldStyle: TextFieldStyle {
    func _body(configuration: TextField<Self._Label>) -> some View {
        configuration
            .padding(12)
            .background(Color.white)
            .cornerRadius(6)
            .overlay(
                RoundedRectangle(cornerRadius: 6)
                    .stroke(Color(red: 0.87, green: 0.87, blue: 0.87), lineWidth: 1)
            )
    }
}

#Preview {
    LoginView()
        .environmentObject(AuthViewModel())
}
