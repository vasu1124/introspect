/*
SPDX-FileCopyrightText: 2024 vasu1124

SPDX-License-Identifier: Apache-2.0
*/
{
  description = "Nix flake for introspect";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs, ... }:
    let
      pname = "introspect";

      # System types to support.
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];

      # Helper function to generate an attrset '{ x86_64-linux = f "x86_64-linux"; ... }'.
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      # Nixpkgs instantiated for supported system types.
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });

    in
    {
      # Add dependencies that are only needed for development
      devShells = forAllSystems (system:
        let 
          pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [ 
              go        # golang
              gopls     # go language server
              gotools   # go imports
              go-tools  # static checks
              gnumake   # standard make

              etcd_3_4
              # mongodb-5_0
              cfssl
              istioctl
              gettext
              jq
              fluxcd
              kubernetes-helm
              kustomize
              krew
              sops
              minikube
              kind
              kubelogin-oidc
              kubectx
              tilt
              kubebuilder
              skaffold
              delve
              ctlptl
              cue
            ];
          };
        });

    };
}
