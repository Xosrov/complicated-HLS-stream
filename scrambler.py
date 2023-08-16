import os
import uuid
import re
import argparse

m3u8FinderRe = re.compile(r'^.*?m3u8$', re.MULTILINE)
tsFinderRe = re.compile(r'^.*?ts$', re.MULTILINE)

def Scramble(directory: str, prefix: str, master_name: str):
    master_manifest_content: str
    # read manifest
    try:
        master_manifest_file = os.path.join(directory, master_name)
        with open(master_manifest_file, 'r') as f:
            master_manifest_content = f.read()
    except Exception:
        print(f"master file not found in {master_manifest_file}")
        exit(1)
    variant_manifests = re.findall(m3u8FinderRe, master_manifest_content)
    if len(variant_manifests) == 0:
        print("no variant manifests found in master manifest")
        exit(1)
    # get location of individual m3u8 files
    for variant_manifest in variant_manifests:
        # read variant
        variant_contents: str
        variant_manifest_file = os.path.join(directory, variant_manifest)
        with open(variant_manifest_file, 'r') as f:
            variant_contents = f.read()
        # rename variant fragment files
        variant_fragments = re.findall(tsFinderRe, variant_contents)
        for variant_fragment in variant_fragments:
            random_fname = uuid.uuid4().__str__()+".ts"
            variant_fragment_filename = os.path.join(directory, variant_fragment)
            os.rename(variant_fragment_filename, os.path.join(directory, random_fname))
            variant_contents = variant_contents.replace(variant_fragment,prefix+random_fname)
        # update variant manifests
        with open(variant_manifest_file, 'w') as f:
            f.write(variant_contents)
        master_manifest_content = master_manifest_content.replace(variant_manifest, prefix+variant_manifest)

    # update master manifest
    with open(os.path.join(directory, master_name), 'w') as f:
        f.write(master_manifest_content)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        prog="TLS Scramblers",
        description="scrable TLS fragments",
    )
    parser.add_argument("-d", "--dir", required=True, help="Directory containing manifests and fragments.")
    parser.add_argument("-p", "--prefix", default="pref:", help="Prefix needed for hls.js, added to manifests. Should end in ':'")
    parser.add_argument("-m", "--master", default="master.m3u8", help="Name of master manifest.")
    args = parser.parse_args()
    Scramble(args.dir, args.prefix, args.master)
