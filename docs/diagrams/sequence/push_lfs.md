# Git LFS Fetch Sequence Diagram

Note: Every response (resp) to git-lfs requests (req) can return an error code and message.

```mermaid
sequenceDiagram
    participant LFS as Git LFS
    participant Helper as Custom Transfer Helper
    participant LocalStorage as Local
    participant Remote as Remote Storage

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
        Helper->>LocalStorage: prepare
        Helper->>LFS: progress (bytesSoFar, bytesSinceLast)
        Helper->>LFS: progress (bytesSoFar, bytesSinceLast)
        Helper-->>LFS: transfer resp (ok)
    end

    %% Termination
    LFS->>Helper: terminate req

   loop (Parallel) For each LFS file
        Helper->>LocalStorage: Copy LFS OCI layer
        LocalStorage->>Remote: Push LFS OCI layer
        Remote-->>LocalStorage: 200 OK
    end

    Helper->>Remote: Push LFS manifest
    Remote-->>Helper: 200 OK
    Helper-->>LFS: terminate resp
```
