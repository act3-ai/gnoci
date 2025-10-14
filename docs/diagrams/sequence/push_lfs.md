# Git LFS Fetch Sequence Diagram

Note: Every response (resp) to git-lfs requests (req) can return an error code and message.

Legend:

- git-lfs: `git-lfs` process.
- git-lfs-remote-oci: `git-lfs-remote-oci` process.
- Local OCI: local OCI storage, prepares LFS files as OCI layers.
- Remote OCI: remote OCI storage, the `oci://<url>` push destination.

```mermaid
sequenceDiagram
    participant LFS as git-lfs
    participant Helper as git-lfs-remote-oci
    participant LocalStorage as Local OCI
    participant Remote as Remote OCI

    Note over LFS,Helper: git-lfs pre-push hook

    %% Initialization (operation, remote name, concurrency options)
    %% op: upload/download
    %% remote: shortname or full URL
    LFS->>Helper: init req (upload, remote, conc)
    Helper-->>LFS: init resp

    Helper->>Remote: fetch manifest
    Remote-->>Helper: manifest

    %% Prepare loop
    loop For each LFS file
        LFS->>Helper: transfer req (upload, OID, path, size)
        Helper->>LocalStorage: Prepare LFS OCI layer
        LocalStorage->>Remote: Push LFS OCI layer
        Helper->>LFS: progress (bytesSoFar, bytesSinceLast)
        Helper->>LFS: progress (bytesSoFar, bytesSinceLast)
        Remote-->>Helper: 200 OK
        Helper-->>LFS: transfer resp (ok)
    end

    Note over LFS,Remote: Complete data model push, Cleanup
    LFS->>Helper: terminate req

    Helper->>Remote: Push LFS manifest
    Remote-->>Helper: 200 OK
    Helper-->>LFS: terminate resp
```
