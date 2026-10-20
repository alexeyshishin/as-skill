---
name: content-documentation
description: >
  Structures a technical tutorial — prerequisites, a "after reading this
  you'll be able to do X" goal, numbered steps with commands and checks,
  troubleshooting, cleanup. Doesn't write the full text: it produces a
  correct skeleton for the author to fill in with details. Use when "write a
  tutorial," "put together step-by-step instructions," "how-to," "tutorial,"
  "deployment guide," "instructions for how to do X."
---

# content-documentation — Technical documentation skeleton

Goal: give the reader a **reproducible** path from their starting state to the goal. Not theory, not an essay — a step-by-step set of instructions where every step is verifiable.

Before starting, read:
- `~/.claude/rules/content-voice.md`
- `~/.claude/rules/content-formatting.md` — the tutorials section

## Step 1. Define the scope

Ask:
- **the tutorial's goal**: "after reading this, the reader will be able to _<verb + result>_"
  - examples: "deploy a Postgres cluster with replication across 3 nodes," "set up OAuth login via Google in an Express app," "measure heap allocations in a Go service"
- **the reader's starting state**: what they have, what's assumed
- **the final state**: what should be working at the end
- **the platform / stack**: versions, OS, cloud

If the goal is vague ("a tutorial on k8s") — **stop and help narrow it down to specifics**. A tutorial "on k8s" is impossible — "deploy minikube + deploy a stateless API + expose via Ingress" is possible.

## Step 2. Build the skeleton

```markdown
# <Title: verb + result>

> **Goal:** after completing this tutorial you'll be able to _<specifically>_.
> **Time:** ~X minutes.
> **Level:** beginner / intermediate / advanced.

## What you'll need

- <tooling: version>
- <tooling: version>
- <access: where to get it>
- <cloud/environment, if applicable>

## Step 1. <Verb + what>

<1-2 lines of context: what we're doing and why.>

```<lang>
<command>
```

**Check:** <expected result / a command confirming success>

## Step 2. <Verb + what>

...

## Step N. <Verb + what>

...

## Verifying the final state

<a command or scenario that verifies everything works end-to-end>

## Troubleshooting

### <Symptom 1>

**Cause:** ...
**Fix:** ...

### <Symptom 2>

...

## Cleanup

<commands to roll back everything that was created — if the reader doesn't want to keep it>

## What's next

<2-3 links: deeper dives, related tutorials, docs>
```

## Step 3. Agree on the skeleton

Show the user **only the structure** (step headings and a brief description of each), without the full text. Ask:
- are all the steps needed?
- is anything missing between steps?
- is the order logical?

Once confirmed — move on to filling it in.

## Step 4. Fill in the steps

For each step:

- **Verb first** in the heading: "Install kubectl," "Create the namespace," "Verify the connection"
- **Context** — why we're doing this (1-2 lines, not a lecture)
- **Command(s)** in a code block with a language tag, **ready to copy-paste** (see `sre-runbook-template.md`)
- **Check** — a command or expected output so the reader can confirm the step succeeded

For every parameter the reader needs to substitute — explain **where to get it**:
```bash
kubectl apply -f deploy.yaml -n <namespace>
# <namespace> — the name from step 2, e.g. "demo-app"
```

## Step 5. Troubleshooting

3-5 common problems people run into. For each:
- **symptom** (how the reader will see it)
- **cause** (why it happens)
- **fix** (what to do)

Don't write troubleshooting entries for hypothetical problems — only for ones that actually happened.

## Step 6. Cleanup

Commands to roll back what was created. This matters — without cleanup the reader is afraid to experiment.

## Step 7. Self-check

- **did you** walk through the tutorial as a first-time reader? Is every step clear without the surrounding article context?
- **are the commands copy-pasteable**? No unexplained `<your-thing>` placeholders?
- **are versions specified**? Six months from now, the tutorial shouldn't break just because a new minor version shipped.
- **is there a check after every step**? Otherwise the reader doesn't know where they went wrong.

## Step 8. Show it and ask

Show the final skeleton with the steps filled in. Ask:
1. Accept
2. Expand a particular step (needs more detail)
3. Compress (the tutorial got long)
4. Add a section (security, production-readiness, alternatives)

## What not to do

- don't write a tutorial for a scenario you've never actually walked through yourself — even partially made-up commands kill trust
- don't explain stack fundamentals inside the tutorial — link to external docs
- don't cram "everything into one article" — a 50+ minute tutorial is better split up
- don't write commands with placeholders like `<your-cluster>` without explaining them
- don't skip the verification section — that's what distinguishes a tutorial from a blog post
