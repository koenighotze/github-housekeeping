# GitHub Housekeeping

GitHub Housekeeping is an agentic development tool that keeps repositories current and healthy with minimal manual effort.

It watches a list of repositories, detects Dependabot updates, pull requests, and CVE-related dependency changes, then takes action according to the policies you define. The tool can review open pull requests, merge safe updates, surface risky changes for human attention, and verify that the main branch still builds after changes are applied.

The goal is to remove the repetitive maintenance work that slows engineering teams down. Instead of chasing update notifications across many repositories, teams get one automated system that scans, prioritizes, validates, and closes routine housekeeping work while preserving control over the changes that matter.

Core capabilities:

- Scan multiple repositories on a schedule or on demand.
- Detect Dependabot pull requests and other automated dependency updates.
- Identify CVE-linked package updates and prioritize them by risk.
- Review pull requests against configurable merge rules.
- Merge safe updates automatically when checks pass.
- Build and verify the main branch after changes are merged.
- Escalate ambiguous or failing cases to a human reviewer.

This product is aimed at teams that want fewer broken dependency updates, faster patch adoption, and a cleaner main branch without spending engineering time on routine repository hygiene.

