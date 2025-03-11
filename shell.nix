with (import (fetchTarball "https://github.com/NixOS/nixpkgs/archive/nixos-24.11.tar.gz") {});

pkgs.mkShell {
	buildInputs = [
		go
	];
}
