/*
 * Copyright © 2015-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * @author		Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @copyright 	2015-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @license 	Apache-2.0
 */

package cli

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ory/hydra/config"
	"github.com/ory/hydra/pkg"
	hydra "github.com/ory/hydra/sdk/go/hydra/swagger"
)

type ClientHandler struct {
	Config *config.Config
}

func newClientHandler(c *config.Config) *ClientHandler {
	return &ClientHandler{
		Config: c,
	}
}

func (h *ClientHandler) newClientManager(cmd *cobra.Command) *hydra.OAuth2Api {
	c := hydra.NewOAuth2ApiWithBasePath(h.Config.GetClusterURLWithoutTailingSlashOrFail(cmd))

	fakeTlsTermination, _ := cmd.Flags().GetBool("skip-tls-verify")
	c.Configuration.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: fakeTlsTermination},
	}

	if term, _ := cmd.Flags().GetBool("fake-tls-termination"); term {
		c.Configuration.DefaultHeader["X-Forwarded-Proto"] = "https"
	}

	if token, _ := cmd.Flags().GetString("access-token"); token != "" {
		c.Configuration.DefaultHeader["Authorization"] = "Bearer " + token
	}

	return c
}

func (h *ClientHandler) ImportClients(cmd *cobra.Command, args []string) {
	m := h.newClientManager(cmd)

	if len(args) == 0 {
		fmt.Print(cmd.UsageString())
		return
	}

	for _, path := range args {
		reader, err := os.Open(path)
		pkg.Must(err, "Could not open file %s: %s", path, err)
		var c hydra.OAuth2Client
		err = json.NewDecoder(reader).Decode(&c)
		pkg.Must(err, "Could not parse JSON: %s", err)

		result, response, err := m.CreateOAuth2Client(c)
		checkResponse(response, err, http.StatusCreated)

		if c.ClientSecret == "" {
			fmt.Printf("Imported OAuth 2.0 Client %s:%s from %s.\n", result.ClientId, result.ClientSecret, path)
		} else {
			fmt.Printf("Imported OAuth 2.0 Client %s from %s.\n", result.ClientId, path)
		}
	}
}

func (h *ClientHandler) CreateClient(cmd *cobra.Command, args []string) {
	var err error
	m := h.newClientManager(cmd)
	responseTypes, _ := cmd.Flags().GetStringSlice("response-types")
	grantTypes, _ := cmd.Flags().GetStringSlice("grant-types")
	allowedScopes, _ := cmd.Flags().GetStringSlice("scope")
	callbacks, _ := cmd.Flags().GetStringSlice("callbacks")
	name, _ := cmd.Flags().GetString("name")
	secret, _ := cmd.Flags().GetString("secret")
	id, _ := cmd.Flags().GetString("id")
	tokenEndpointAuthMethod, _ := cmd.Flags().GetString("token-endpoint-auth-method")
	jwksUri, _ := cmd.Flags().GetString("jwks-uri")
	tosUri, _ := cmd.Flags().GetString("tos-uri")
	policyUri, _ := cmd.Flags().GetString("policy-uri")
	logoUri, _ := cmd.Flags().GetString("logo-uri")
	clientUri, _ := cmd.Flags().GetString("client-uri")
	subjectType, _ := cmd.Flags().GetString("subject-type")

	var echoSecret bool

	if secret == "" {
		var secretb []byte
		secretb, err = pkg.GenerateSecret(26)
		pkg.Must(err, "Could not generate OAuth 2.0 Client Secret: %s", err)
		secret = string(secretb)

		echoSecret = true
	} else {
		fmt.Println("You should not provide secrets using command line flags. The secret might leak to bash history and similar systems.")
	}

	cc := hydra.OAuth2Client{
		ClientId:                id,
		ClientSecret:            secret,
		ResponseTypes:           responseTypes,
		Scope:                   strings.Join(allowedScopes, " "),
		GrantTypes:              grantTypes,
		RedirectUris:            callbacks,
		ClientName:              name,
		TokenEndpointAuthMethod: tokenEndpointAuthMethod,
		JwksUri:                 jwksUri,
		TosUri:                  tosUri,
		PolicyUri:               policyUri,
		LogoUri:                 logoUri,
		ClientUri:               clientUri,
		SubjectType:             subjectType,
	}

	result, response, err := m.CreateOAuth2Client(cc)
	checkResponse(response, err, http.StatusCreated)

	fmt.Printf("OAuth 2.0 Client ID: %s\n", result.ClientId)

	if result.ClientSecret == "" {
		fmt.Println("This OAuth 2.0 Client has no secret.")
	} else {
		if echoSecret {
			fmt.Printf("OAuth 2.0 Client Secret: %s\n", result.ClientSecret)
		}
	}
}

func (h *ClientHandler) DeleteClient(cmd *cobra.Command, args []string) {
	m := h.newClientManager(cmd)

	if len(args) == 0 {
		fmt.Print(cmd.UsageString())
		return
	}

	for _, c := range args {
		response, err := m.DeleteOAuth2Client(c)
		checkResponse(response, err, http.StatusNoContent)
	}

	fmt.Println("OAuth2 client(s) deleted.")
}

func (h *ClientHandler) GetClient(cmd *cobra.Command, args []string) {
	m := h.newClientManager(cmd)

	if len(args) == 0 {
		fmt.Print(cmd.UsageString())
		return
	}

	cl, response, err := m.GetOAuth2Client(args[0])
	checkResponse(response, err, http.StatusOK)
	fmt.Printf("%s\n", formatResponse(cl))
}
