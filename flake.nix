{
  inputs = {
    # nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    systems.url = "github:nix-systems/default";
  };

  outputs = {
    systems,
    nixpkgs,
    ...
  } @ inputs: let
    eachSystem = f:
      nixpkgs.lib.genAttrs (import systems) (
        system:
          f nixpkgs.legacyPackages.${system}
      );
    networkinfo = pkgs: pkgs.buildGoModule {
      pname = "networkinfo";
      version = "0.1.0";
      src = ./.;
      vendorHash = null;
    };
  in {
    packages = eachSystem (pkgs: let nwinfo = networkinfo pkgs; in {
      sway-networkinfo = nwinfo;
      default = nwinfo;
    });

    # devShells = eachSystem (pkgs: {
    #   default = pkgs.mkShell {
    #     buildInputs = with pkgs; [
    #       # Add development dependencies here
    #     ];
    #   };
    # });
  };
}
