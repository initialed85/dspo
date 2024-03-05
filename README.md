# dspo

# status: not yet working

A simple tool to help you run some processes in the background; `dspo` is short for `dead-simple-process-orchestrator`
and provides some sugar as long as your problem fits the following pattern:

-   You need to run some processes in the background
-   Some of those processes themselves may be background processes
-   You want to pull up the logs from time to time
-   You want to be able to take down all the processes at once
-   Using `docker compose` makes sense to you

## Concept

-   The user describes on or more `.yaml` files (default `dspo.yaml`)
-   The user (optionally) describes a `.env` file
-   The user runs `dspo up` or `dspo up -d` to start the processes
-   The user can run `dspo logs` or `dspo logs -f` to see the logs
-   The user can run `dspo down` to stop the processes

## Notes

### Fundamentals

-   `Process`
    -   Minimal abstraction around a single execution of a process
-   `ManagedProcess`
    -   Adds the lifecycle management (restarts, etc)
-   ## `Service`

### Other

-   `Fanout`
    -   Single-producer multi-consumer fanout for channels
-   `Probe`
    -   Instance of `ManagedProcess` to probe for startup / liveness
