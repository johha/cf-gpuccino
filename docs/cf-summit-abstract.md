# Your Prompts, Your Platform: GPU Workloads in Cloud Foundry

> Draft abstract for Cloud Foundry Summit 2026 (Heidelberg).
> Character limit: 1200.

---

## Abstract

AI workloads usually leave the platform. A prompt goes out to OpenAI or
Anthropic, an answer comes back. That works for most cases, but not all of
them: sensitive data that can't leave the building, token bills that grow
faster than the value, private clouds with no path to the public internet.
And plenty of GPU work isn't even chat-shaped, think local inference,
embeddings, image and video processing.

This isn't about replacing the big AI providers. It's about the workloads
that belong on the platform you already run.

Kubernetes solved this years ago. Cloud Foundry has stayed CPU-only. So we wondered what it would actually take to change that, and started building.

The honest answer is: a lot. GPU support reaches into the BOSH stemcell, the
NVIDIA driver and Container Device Interface, garden-runc and the OCI spec,
Diego's auctioneer and cell rep, the CAPI process model, and finally the CLI
and app manifest. Almost every layer of CF has something to say about it.

This talk walks through what we built, what surprised us, and the questions
we still don't have answers to.
