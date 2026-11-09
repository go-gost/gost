#!/usr/bin/env bash

if [ -n "${ZSH_VERSION-}" ]; then
    printf 'error: this is a bash script; run it as: bash %s\n' "$0" >&2
    exit 1
fi

if [ -z "${BASH_VERSION-}" ]; then
    printf 'error: this installer requires bash (>= 3.2)\n' >&2
    printf '  Debian/Ubuntu: apt install bash\n  Alpine: apk add bash\n  FreeBSD: pkg install bash\n' >&2
    printf '  or download a release binary: https://github.com/go-gost/gost/releases\n' >&2
    printf '  Windows: run this in Git Bash (POSIX shell), not cmd.exe or PowerShell\n' >&2
    exit 1
fi

set -euo pipefail
umask 022

usage() {
    cat <<'EOF'
Usage: install.sh [OPTION]...

Options:
  --install             install the latest release (non-interactive)
  --version <version>   install a specific release, e.g. v3.3.3
  --list                list available release tags and exit
  --print-asset         print the resolved release asset name and exit
  --prefix <dir>        install into <dir>/bin
  --skip-verify         skip checksum verification (not recommended)
  --dry-run             resolve and print the plan, then exit
  -v, --verbose         trace the download/verify/install steps
  --cpu-v3              prefer the amd64v3 build on x86_64
  --mips-float <abi>    mips float ABI: soft (default) or hard
  --mips-endian <order> mips64 byte order: be (default) or le
  -h, --help            show this help

Environment:
  PREFIX                same as --prefix

Defaults: root installs into /usr/local, non-root into ~/.local
EOF
}

set_mips_float() {
    case $1 in
    soft | hard) mips_float=$1 ;;
    *)
        printf 'error: --mips-float must be soft or hard\n' >&2
        exit 2
        ;;
    esac
}

set_mips_endian() {
    case $1 in
    be | le) mips_endian=$1 ;;
    *)
        printf 'error: --mips-endian must be be or le\n' >&2
        exit 2
        ;;
    esac
}

mode=interactive
version_arg=
prefix_arg=
print_asset=0
cpu_v3=0
mips_float=soft
mips_endian=be
skip_verify=0
dry_run=0
verbose=0

while [ $# -gt 0 ]; do
    case $1 in
    --install)
        mode=latest
        shift
        ;;
    --version)
        [ $# -ge 2 ] || {
            usage >&2
            exit 2
        }
        mode=version
        version_arg=$2
        shift 2
        ;;
    --version=*)
        mode=version
        version_arg=${1#*=}
        shift
        ;;
    --list)
        mode=list
        shift
        ;;
    --prefix)
        [ $# -ge 2 ] || {
            usage >&2
            exit 2
        }
        prefix_arg=$2
        shift 2
        ;;
    --prefix=*)
        prefix_arg=${1#*=}
        shift
        ;;
    --skip-verify)
        skip_verify=1
        shift
        ;;
    --dry-run)
        dry_run=1
        shift
        ;;
    -v | --verbose)
        verbose=1
        shift
        ;;
    --print-asset)
        print_asset=1
        shift
        ;;
    --cpu-v3)
        cpu_v3=1
        shift
        ;;
    --mips-float)
        [ $# -ge 2 ] || {
            usage >&2
            exit 2
        }
        set_mips_float "$2"
        shift 2
        ;;
    --mips-float=*)
        set_mips_float "${1#*=}"
        shift
        ;;
    --mips-endian)
        [ $# -ge 2 ] || {
            usage >&2
            exit 2
        }
        set_mips_endian "$2"
        shift 2
        ;;
    --mips-endian=*)
        set_mips_endian "${1#*=}"
        shift
        ;;
    -h | --help)
        usage
        exit 0
        ;;
    --)
        shift
        break
        ;;
    -*)
        printf 'error: unknown option: %s\n' "$1" >&2
        usage >&2
        exit 2
        ;;
    *)
        printf 'error: unexpected argument: %s\n' "$1" >&2
        usage >&2
        exit 2
        ;;
    esac
done

detect_os() {
    case $(uname) in
    Linux) os=linux ;;
    Darwin) os=darwin ;;
    FreeBSD) os=freebsd ;;
    MINGW* | MSYS* | CYGWIN*) os=windows ;;
    *) return 1 ;;
    esac

    if [ "$os" = windows ]; then
        ext=zip
    else
        ext=tar.gz
    fi
}

detect_arch() {
    # 顺序敏感：mips64* 必须在 mips* 之前，mipsel* 也必须在 mips* 之前
    case $1 in
    x86_64 | amd64)
        if [ "$cpu_v3" = 1 ]; then
            cpu_arch=amd64v3
        else
            cpu_arch=amd64
        fi
        ;;
    aarch64 | arm64) cpu_arch=arm64 ;;
    armv5*) cpu_arch=armv5 ;;
    armv6*) cpu_arch=armv6 ;;
    armv7* | armv8l) cpu_arch=armv7 ;;
    i386 | i486 | i586 | i686) cpu_arch=386 ;;
    riscv64) cpu_arch=riscv64 ;;
    s390x) cpu_arch=s390x ;;
    loongarch64 | loong64) cpu_arch=loong64 ;;
    mips64el | mips64le) cpu_arch=mips64le_hardfloat ;;
    mips64)
        if [ "$mips_endian" = le ]; then
            cpu_arch=mips64le_hardfloat
        else
            cpu_arch=mips64_hardfloat
        fi
        ;;
    mipsel | mipsle)
        if [ "$mips_float" = hard ]; then
            cpu_arch=mipsle_hardfloat
        else
            cpu_arch=mipsle_softfloat
        fi
        ;;
    mips)
        if [ "$mips_float" = hard ]; then
            cpu_arch=mips_hardfloat
        else
            cpu_arch=mips_softfloat
        fi
        ;;
    *) return 1 ;;
    esac
}

asset_name() {
    # $1: version with or without leading v
    printf 'gost_%s_%s_%s.%s' "${1#v}" "$os" "$cpu_arch" "$ext"
}

if [ "$print_asset" = 1 ]; then
    detect_os || {
        printf 'error: unsupported OS: %s\n' "$(uname)" >&2
        exit 1
    }
    detect_arch "$(uname -m)" || {
        printf 'error: unsupported architecture: %s\n' "$(uname -m)" >&2
        exit 1
    }
    if [ -n "$version_arg" ]; then
        printf '%s\n' "$(asset_name "$version_arg")"
    else
        printf '%s_%s.%s\n' "$os" "$cpu_arch" "$ext"
    fi
    exit 0
fi

require() {
    command -v "$1" >/dev/null 2>&1 || {
        printf 'error: required command not found: %s\n' "$1" >&2
        exit 1
    }
}

require curl
require tar
require awk
require uname

# Set the desired GitHub repository
repo="go-gost/gost"
base_url="https://api.github.com/repos/$repo/releases"
download_base="https://github.com/$repo/releases/download"

sha256_file() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{ print $1 }'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{ print $1 }'
    elif command -v certutil >/dev/null 2>&1; then
        certutil -hashfile "$1" SHA256 | sed -n '2p' | tr -d ' \r' | tr '[:upper:]' '[:lower:]'
    else
        printf 'error: no SHA-256 tool found (sha256sum / shasum / certutil)\n' >&2
        return 1
    fi
}

checksum_for() {
    # $1=version(without v) $2=asset name -> prints expected hash
    url="$download_base/v$1/checksums.txt"
    body=$(curl -fsSL --retry 3 "$url") || {
        printf 'error: cannot fetch checksums: %s\n' "$url" >&2
        return 1
    }
    hash=$(printf '%s\n' "$body" | awk -v f="$2" '{ n=$2; sub(/^\*/, "", n); if (n == f) { print $1; exit } }')
    [ -n "$hash" ] || {
        printf 'error: no checksum for %s in %s\n' "$2" "$url" >&2
        return 1
    }
    printf '%s\n' "$hash" | tr '[:upper:]' '[:lower:]'
}

verify_asset() {
    # $1=file $2=asset name $3=version(without v)
    if [ "$skip_verify" = 1 ]; then
        printf 'warning: skipping checksum verification for %s\n' "$2" >&2
        return 0
    fi
    expected=$(checksum_for "$3" "$2") || return 1
    actual=$(sha256_file "$1") || return 1
    if [ "$actual" != "$expected" ]; then
        printf 'error: checksum mismatch for %s\n  expected: %s\n  actual:   %s\n' "$2" "$expected" "$actual" >&2
        return 1
    fi
    printf 'Checksum OK: %s\n' "$2"
}

print_plan() {
    # $1=version(with v) $2=asset $3=url $4=posix|windows
    printf 'dry-run: no changes will be made\n'
    printf '  version:   %s\n' "$1"
    printf '  platform:  %s/%s\n' "$os" "$cpu_arch"
    printf '  asset:     %s\n' "$2"
    printf '  url:       %s\n' "$3"

    hash=$(checksum_for "${1#v}" "$2" 2>/dev/null) || hash='(unavailable)'
    printf '  checksum:  %s\n' "$hash"

    if [ "$4" = windows ]; then
        printf '  save to:   %s\n' "${HOME:?HOME is not set}/Downloads/$2"
        printf '  actions:   download -> verify -> reveal in Explorer\n'
    else
        printf '  install:   %s\n' "$install_path/gost"
        printf '  actions:   download -> verify -> extract -> install -> gost -V\n'
    fi
}

# Function to download and install gost
install_windows() {
    version=$1
    download_url=$2

    # 非 POSIX shell（没有 cygpath）→ 直接失败，不做兼容
    if ! command -v cygpath >/dev/null 2>&1; then
        printf 'error: cygpath not found; run this in Git Bash, not cmd.exe/PowerShell\n' >&2
        printf '  download manually: %s\n' "$download_url" >&2
        exit 1
    fi

    dir="${HOME:?HOME is not set}/Downloads"
    ver=${version#v}

    if [ "$dry_run" = 1 ]; then
        print_plan "$version" "$(asset_name "$ver")" "$download_url" windows
        exit 0
    fi

    if ! mkdir -p "$dir" 2>/dev/null; then
        printf 'error: cannot create %s\n' "$dir" >&2
        printf '  download manually: %s\n' "$download_url" >&2
        exit 1
    fi

    zip="$dir/$(asset_name "$ver")"

    if [ "$verbose" = 1 ]; then set -x; fi
    printf 'Downloading %s\n' "$(asset_name "$ver")"
    if ! curl -fsSL --retry 3 -o "$zip" "$download_url"; then
        printf 'error: download failed: %s\n' "$download_url" >&2
        exit 1
    fi

    if ! verify_asset "$zip" "$(asset_name "$ver")" "$ver"; then
        rm -f "$zip"
        exit 1
    fi
    if [ "$verbose" = 1 ]; then set +x; fi

    winzip=$(cygpath -w "$zip")
    printf '\nSaved: %s\n\n' "$winzip"
    printf 'Extract gost.exe from the zip, then run:\n'
    printf '  gost.exe -L "http://:8080"\n'
    printf 'Note: this script does not install a service.\n'

    explorer_bin=$(command -v explorer.exe || true)
    if [ -z "$explorer_bin" ]; then
        explorer_bin=$(cygpath -u "${SYSTEMROOT:-C:\\Windows}")/explorer.exe
    fi
    if [ ! -x "$explorer_bin" ]; then
        printf 'error: explorer.exe not found; the file is here: %s\n' "$winzip" >&2
        exit 1
    fi
    "$explorer_bin" "/select,$winzip" >/dev/null 2>&1 || true
}

install_gost() {
    version=$1

    if [ -r /proc/version ] && grep -qi microsoft /proc/version; then
        printf 'note: WSL detected; installing the Linux binary (use Git Bash for a Windows install).\n' >&2
    fi
    detect_os || {
        printf 'error: unsupported OS: %s\n' "$(uname)" >&2
        printf '  supported: linux, darwin, freebsd, windows (Git Bash/MSYS/Cygwin)\n' >&2
        exit 1
    }

    detect_arch "$(uname -m)" || {
        printf 'error: unsupported architecture: %s\n' "$(uname -m)" >&2
        printf '  supported: amd64, arm64, armv5-7, 386, riscv64, s390x, loong64, mips(le/64/64le)\n' >&2
        exit 1
    }
    [ -n "$version" ] || {
        printf 'error: version is empty\n' >&2
        exit 1
    }
    ver=${version#v}
    asset=$(asset_name "$ver")
    download_url="$download_base/v$ver/$asset"

    if [ "$os" = windows ]; then
        install_windows "$version" "$download_url"
        return 0
    fi

    if [ -n "$prefix_arg" ]; then
        prefix=$prefix_arg
    elif [ -n "${PREFIX-}" ]; then
        prefix=$PREFIX
    elif [ "$(id -u)" -eq 0 ]; then
        prefix=/usr/local
    else
        prefix=${HOME:?HOME is not set}/.local
    fi

    install_path="$prefix/bin"

    if [ "$dry_run" = 1 ]; then
        print_plan "$version" "$asset" "$download_url" posix
        exit 0
    fi

    if ! mkdir -p "$install_path" 2>/dev/null; then
        printf 'error: cannot create %s\n' "$install_path" >&2
        printf '  pick a writable prefix: PREFIX=/some/dir %s\n' "$0" >&2
        printf '  system-wide installs need root: sudo PREFIX=/usr/local %s\n' "$0" >&2
        exit 1
    fi

    if [ ! -w "$install_path" ]; then
        printf 'error: %s is not writable\n' "$install_path" >&2
        printf '  use PREFIX=/some/writable/dir (or sudo for a system-wide install)\n' >&2
        exit 1
    fi

    # Download, verify, then install
    if [ "$verbose" = 1 ]; then set -x; fi

    tmpdir=$(mktemp -d) || {
        printf 'error: mktemp failed\n' >&2
        exit 1
    }
    trap 'rm -rf "$tmpdir"' EXIT

    asset_file="$tmpdir/$asset"

    printf 'Downloading %s\n' "$asset"
    if ! curl -fsSL --retry 3 -o "$asset_file" "$download_url"; then
        printf 'error: download failed: %s\n' "$download_url" >&2
        exit 1
    fi

    verify_asset "$asset_file" "$asset" "$ver" || exit 1

    tar -xzf "$asset_file" -C "$tmpdir" gost || {
        printf 'error: extract failed: %s\n' "$asset_file" >&2
        exit 1
    }

    tmpbin="$install_path/.gost.tmp.$$"
    if ! install -m 0755 "$tmpdir/gost" "$tmpbin"; then
        rm -f "$tmpbin"
        printf 'error: cannot write %s\n' "$install_path" >&2
        exit 1
    fi
    if ! mv -f "$tmpbin" "$install_path/gost"; then
        rm -f "$tmpbin"
        printf 'error: cannot replace %s/gost\n' "$install_path" >&2
        exit 1
    fi

    installed=$("$install_path/gost" -V 2>&1) || {
        printf 'error: installed gost does not run: %s\n' "$installed" >&2
        exit 1
    }
    case $installed in
    "gost $ver "*) ;;
    *)
        printf 'warning: installed version does not match %s: %s\n' "$ver" "$installed" >&2
        ;;
    esac

    if [ "$verbose" = 1 ]; then set +x; fi

    # Remove binary from macOS quarantine when installing for first time
    if [ "$os" = darwin ] && command -v xattr >/dev/null 2>&1; then
        xattr -d com.apple.quarantine "$install_path/gost" 2>/dev/null || true
    fi

    echo "gost installation completed!"

    case ":$PATH:" in
    *":$install_path:"*) ;;
    *)
        printf '\n%s is not in PATH. Add it:\n' "$install_path"
        printf '  bash: echo '\''export PATH="%s:$PATH"'\'' >> ~/.bashrc\n' "$install_path"
        printf '  zsh:  echo '\''export PATH="%s:$PATH"'\'' >> ~/.zshrc\n' "$install_path"
        ;;
    esac
}

fetch_versions() {
    raw=$(curl -sS -w '\n%{http_code}' "$base_url") || {
        printf 'error: could not fetch release list\n' >&2
        return 1
    }
    code=${raw##*$'\n'}
    body=${raw%$'\n'*}

    case $code in
    [0-9][0-9][0-9]) ;;
    *)
        printf 'error: could not fetch release list\n' >&2
        return 1
        ;;
    esac

    case $code in
    200)
        printf '%s\n' "$body" | awk -F'"' '/"tag_name":/ { print $4 }'
        ;;
    403 | 429)
        printf 'error: GitHub API rate limit or rejection (HTTP %s)\n' "$code" >&2
        printf '  retry later, or use --version <tag>\n' >&2
        return 1
        ;;
    *)
        printf 'error: GitHub API returned HTTP %s\n' "$code" >&2
        return 1
        ;;
    esac
}

require_release_list() {
    versions=$(fetch_versions) || exit 1
    [ -n "$versions" ] || {
        printf 'error: no releases found\n' >&2
        exit 1
    }
}

if [ "$mode" = list ]; then
    require_release_list
    printf '%s\n' "$versions"
    exit 0
fi

if [ "$mode" != version ]; then
    require_release_list
fi

case $mode in
version)
    install_gost "$version_arg"
    ;;
latest)
    first_version=${versions%%$'\n'*}
    install_gost "$first_version"
    ;;
interactive)
    echo "Available gost versions:"
    # shellcheck disable=SC2086  # intentional: one menu item per tag
    select version in $versions; do
        if [ -n "$version" ]; then
            install_gost "$version"
            break
        fi
        echo "Invalid choice! Please select a valid option."
    done
    ;;
esac
