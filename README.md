# Playground

A polyglot monorepo and digital space for learning, experimenting and practising smaller software concepts.

## Architecture Diagram

```mermaid
graph TD
    %% Infrastructure Layer
    subgraph Infrastructure [Infrastructure - deed]
        direction TB
        AWS[AWS - eu-west-2]
        S3[S3 Buckets]
        CF[CloudFront Distribution]
        DNS[DNS]
        IAM[IAM Roles]
    end

    %% Core Application Components
    subgraph Core [Application Components]
        direction TB
        custodian[custodian - API / Persistence]
        broom[broom - CLI / Content Editor]
        persona[persona - Public Website]
        blank[blank - UI Library]
    end

    %% Relationships
    deed -- "provisions" --> AWS
    deed -- "provisions" --> S3
    deed -- "provisions" --> CF
    deed -- "provisions" --> DNS
    deed -- "provisions" --> IAM

    broom -- "writes / configures" --> custodian
    custodian -- "serves content to" --> persona
    persona -- "consumes" --> blank
    
    %% Media Flow
    custodian -- "presigns / manages" --> S3
    CF -- "serves media from" --> S3
    persona -- "requests media via CDN" --> CF

    %% Content Types
    subgraph Content [Data Model]
        direction LR
        Authored[Authored Content: logs, media]
        Derived[Derived Content: integrations]
        Profile[Profile Content: about, skills, etc.]
    end

    custodian --- Authored
    custodian --- Derived
    custodian --- Profile
    persona -. "bakes at build time" .-> Profile
    persona -. "fetches at runtime" .-> Authored
    persona -. "fetches at runtime" .-> Derived
```

## Components

- **`custodian`**: The API that keeps the playground. It owns all content, serves it to `persona` at runtime, accepts writes from `broom`, and is the only component holding third-party credentials.
- **`broom`**: The command-line tool used to configure, edit and write content. It is a client of `custodian`'s API and has no storage or authority of its own.
- **`blank`**: The Web Components UI library. Consumed by `persona`.
- **`persona`**: The public website. Bakes profile content at build time and loads blogs and live status from `custodian` at runtime. Built with `blank`.
- **`deed`**: The infrastructure-as-code component. Provisions the EC2 instance, S3 buckets, CloudFront distribution, IAM roles, and DNS.
