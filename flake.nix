{
  description = "TODO";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:

    {
      nixosModule = ({ pkgs, ... }: {
        imports = [ ./default.nix ];
        nixpkgs.overlays =
          [ (_self: _super: { s3photoalbum = self.packages.${pkgs.system}.s3photoalbum; }) ];
      });

    } //

    flake-utils.lib.eachDefaultSystem (system:
      with nixpkgs.legacyPackages.${system}; rec {

        packages = flake-utils.lib.flattenTree rec {

          s3photoalbum = buildGoModule rec {

            pname = "s3photoalbum";
            version = "0.1";

            src = ./.;
            vendorSha256 = "sha256-HcPUWpPpiwBf3wUHu1Mh6Jk3FfMtsoLhXXg0RoABUno=";
            subPackages = [ "cmd/server" "cmd/thumbnailer" ];
            installPhase = ''
              mkdir -p $out/share
              cp -r /build/go/bin $out
              cp -r ./templates $out/share/
              cp -r ./static $out/share/
              '';


            meta = with lib; {
              description = "TODO";
              homepage = "https://github.com/pinpox/s3photoalbum";
              license = licenses.gpl3;
              maintainers = with maintainers; [ pinpox ];
              platforms = platforms.linux;
            };
          };
        };

        defaultPackage = packages.s3photoalbum;
      });
}
