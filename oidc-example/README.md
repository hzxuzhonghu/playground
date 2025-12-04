### 1. Run the OIDC Server:

  Open a terminal and run the following command:

   1 go run server/main.go

  You should see the output: OIDC Server started on http://localhost:8080

### 2. Run the OIDC Client:

  Open a second terminal and run the following command:

   1 go run client/main.go

  You should see the output: OIDC Client started on http://localhost:8081

### 3. Test the Authentication Flow:

   1. Open your web browser and navigate to the client application at http://localhost:8081.
   2. Click the "Login" link.
   3. You will be redirected to the OIDC server's login page.
   4. Enter the following credentials:
       * Username: user
       * Password: password
   5. Click "Login".
   6. You will be redirected back to the client application, which will display the ID token's claims in JSON format.

  This demonstrates a complete OIDC authentication flow, from login to token verification.
