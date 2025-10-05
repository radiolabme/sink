# Why Sink Will Fail: Learning from Dead Projects

**Date:** October 4, 2025  
**Purpose:** Honest assessment of failure modes to avoid them

---

## The Graveyard: Projects That Tried and Died

### 1. **Boxen (GitHub, 2012-2016)** 💀

**What it was:**
- Mac dev environment automation via Puppet
- Backed by GitHub (huge credibility)
- Solved real GitHub employee pain

**Why it died:**
```
"Boxen is no longer maintained by GitHub. We recommend using 
a different solution for managing your development environment."
```

**The failure modes:**

#### A. **Complexity Creep**
- Started simple
- Grew to require: Ruby, Puppet, Xcode CLI tools, homebrew
- Setup script was 200+ lines
- **Lesson:** Dependencies kill adoption

#### B. **Maintenance Burden**
- Puppet modules constantly broke
- macOS updates broke everything
- Required dedicated team
- **Lesson:** External dependencies = constant breakage

#### C. **Wrong Abstraction**
- Puppet is for servers, not laptops
- Mac users don't think in "manifests"
- **Lesson:** Tool must match mental model

#### D. **Scope Creep**
- Tried to manage dotfiles
- Tried to manage projects
- Tried to manage everything
- **Lesson:** Do one thing well

**What you're doing wrong if Sink follows this path:**
- ✅ You're avoiding dependencies (good)
- ⚠️ Risk: Plugin system could become "new Puppet modules"
- ⚠️ Risk: Config schema could grow too complex
- ⚠️ Risk: Trying to solve too many problems

---

### 2. **Otto (HashiCorp, 2015-2016)** 💀

**What it was:**
- "Successor to Vagrant"
- Development environment automation
- Auto-detect and configure

**Why it died (in 12 months):**
```
"We will not be actively developing Otto going forward."
```

**The failure modes:**

#### A. **Magic Over Control**
- Tried to auto-detect everything
- Users couldn't understand what it did
- Debugging was impossible
- **Lesson:** Explicit > Implicit

#### B. **Competing Products**
- Overlapped with Vagrant
- Overlapped with Terraform
- Unclear positioning
- **Lesson:** Must have clear unique value

#### C. **Too Ambitious**
- Tried to handle: detection, build, deploy, dev, infra
- Couldn't do any one thing excellently
- **Lesson:** Narrow scope wins

**What you're doing wrong if Sink follows this path:**
- ✅ Explicit configs (good)
- ⚠️ Risk: Facts system could become "magic detection"
- ⚠️ Risk: Overlapping with Make/Just/Ansible
- ⚠️ Risk: Unclear when to use Sink vs. alternatives

---

### 3. **Chef Solo → Chef Zero → Chef Workstation** 💀

**What it was:**
- "Lightweight" Chef for single machines
- No server required
- Same DSL as Chef Server

**Why it failed:**

#### A. **Ruby Dependency**
- Required Ruby runtime
- Version conflicts
- Slow startup
- **Lesson:** Runtime dependencies are poison

#### B. **Complexity Inherited**
- Carried Chef Server complexity
- Cookbooks were still complex
- Learning curve too steep
- **Lesson:** Can't simplify by removing features

#### C. **Identity Crisis**
- Not as simple as scripts
- Not as powerful as Chef Server
- Worst of both worlds
- **Lesson:** Must be better at something specific

**What you're doing wrong if Sink follows this path:**
- ✅ No runtime dependencies (good)
- ✅ Simple config format (good)
- ⚠️ Risk: Plugin architecture could inherit complexity
- ⚠️ Risk: Trying to compete with Ansible on features

---

### 4. **Sprinkle (2008-2012)** 💀

**What it was:**
- Ruby DSL for server provisioning
- "Like Capistrano for setup"
- Simple, elegant

**Why it died:**

#### A. **DSL Fatigue**
- Another DSL to learn
- Ruby knowledge required
- Limited to what DSL allowed
- **Lesson:** DSLs are maintenance nightmares

#### B. **Community Size**
- Small community
- Few contributors
- Packages outdated
- **Lesson:** Need critical mass

#### C. **Overtaken by Better Tools**
- Ansible came out (simpler)
- Chef/Puppet more mature
- No compelling reason to switch
- **Lesson:** Must be 10x better at something

**What you're doing wrong if Sink follows this path:**
- ✅ JSON not DSL (good)
- ⚠️ Risk: Small community, no adoption
- ⚠️ Risk: Not 10x better than alternatives
- ⚠️ Risk: Config format could become DSL-like

---

### 5. **Babushka (2010-2015)** 💀

**What it was:**
- "Test-driven system admin"
- Ruby-based dependency management
- Clever "deps" system

**Why it died:**

#### A. **Too Clever**
```ruby
dep 'postgres' do
  requires 'postgres.managed'
  met? { shell? "psql --version" }
end
```
- Mental overhead
- Hard to debug
- **Lesson:** Clever is not user-friendly

#### B. **Dependency Hell**
- Deps depended on deps
- Circular deps possible
- Hard to reason about
- **Lesson:** Keep dependency graph simple

#### C. **Niche Appeal**
- Only Ruby devs adopted it
- Too programmer-centric
- **Lesson:** Target broader audience

**What you're doing wrong if Sink follows this path:**
- ✅ Simple check-then-act (good)
- ⚠️ Risk: Facts + templates could get "too clever"
- ⚠️ Risk: Targeting only Go/programmer types
- ⚠️ Risk: Making configs too programmer-centric

---

## The Pattern: Why They ALL Failed

### Common Thread #1: **Complexity Creep**

**The trajectory:**
```
Version 0.1: Simple, focused, beautiful
Version 0.5: Added features users asked for
Version 1.0: Now has config for the config
Version 2.0: Requires manual to use
Version 3.0: Project abandoned
```

**How it happens:**
1. User asks for feature X
2. You add it (seems reasonable)
3. Feature X needs configuration
4. Configuration needs validation
5. Validation needs error messages
6. Error messages need i18n
7. i18n needs... (death spiral)

**How to avoid:**
- **Say NO to features**
- **Every feature must justify its complexity**
- **Measure complexity budget** (LOC, concepts, config options)
- **Remove features regularly** (controversial!)

---

### Common Thread #2: **Wrong Abstraction Level**

**The mistake:**
```
Too Low:  Just wraps shell commands (why not bash?)
Too High: Requires learning new concepts (why not Ansible?)
```

**The sweet spot:**
```
Platform abstraction: YES (darwin/linux/windows)
Command abstraction: NO (shell commands are fine)
Package abstraction: NO (let users use apt/brew/etc)
Service abstraction: NO (let users use systemd/launchd)
```

**Your current design:**
- ✅ Platform abstraction (good)
- ✅ Bare shell commands (good)
- ✅ No package manager abstraction (good)
- ⚠️ Risk: Plugin system adds too many abstractions

---

### Common Thread #3: **Dependency Trap**

**The death spiral:**
```
Project requires: Python
User has Python 2.7, project needs 3.8
User installs Python 3.8
Now Ansible breaks (needs 2.7)
User gives up, writes bash script
```

**Your current approach:**
- ✅ Zero dependencies (Go stdlib only)
- ✅ Single binary
- ⚠️ Risk: Plugins might need dependencies
- ⚠️ Risk: Facts might need external tools

**Critical rule:**
> **If Sink ever requires anything beyond the stdlib, it has failed.**

---

### Common Thread #4: **Network Effects Failure**

**The chicken-and-egg:**
```
No users → No shared configs → No reason to adopt
Few users → Few contributors → Project stagnates
Stagnant project → Users leave → Death spiral
```

**How projects bootstrap:**
- ✅ Solve YOUR pain first (you're doing this)
- ✅ Share configs that solve common problems
- ✅ Make it trivially easy to share
- ⚠️ Risk: Config format too complex to share
- ⚠️ Risk: No ecosystem/marketplace

**Critical path:**
1. Get 10 power users
2. They create 100 configs
3. Those configs attract 1000 users
4. Network effects kick in

---

### Common Thread #5: **Maintenance Burden**

**Why maintainers burn out:**
```
Year 1: Excited, lots of commits
Year 2: Feature requests pile up
Year 3: Breaking changes needed but can't
Year 4: Maintenance mode
Year 5: Archived
```

**Factors:**
- Supporting old versions
- Backward compatibility
- Bug reports for edge cases
- Documentation decay
- Plugin maintenance

**How to avoid:**
- **Small core** (easier to maintain)
- **Clear deprecation policy** (can break things)
- **Stable config format** (v1.0 works forever)
- **No official plugins** (community maintains)

---

## The Brutal Truth: Why Sink WILL Fail

Let me be completely honest about the most likely failure modes:

### Failure Mode #1: **Nobody Uses It** (80% probability)

**Why:**
- Another tool to learn
- Bash scripts "work" (good enough)
- Network effects of existing tools
- Documentation not good enough
- Onboarding too complex

**How to know you're failing:**
- 6 months: < 10 GitHub stars
- 1 year: < 100 stars
- 2 years: Only you use it

**How to prevent:**
- Ship with 20 example configs
- Make onboarding 30 seconds
- Target a specific pain point
- Build for specific community first

---

### Failure Mode #2: **Complexity Explosion** (60% probability)

**Why:**
- Plugin system becomes "new modules"
- Config format grows features
- Backward compatibility constraints
- Edge cases pile up

**How to know you're failing:**
- Config files > 200 lines common
- Users ask "how do I..." constantly
- PRs add config options
- Core LOC > 3000

**How to prevent:**
- **Hard LOC limit: 2000**
- **Config option limit: 20 total**
- **Remove features yearly**
- **Say NO to 90% of requests**

---

### Failure Mode #3: **Wrong Problem** (40% probability)

**Why:**
- You're solving YOUR problem (n=1)
- Others don't have this problem
- Or they solve it differently
- Or they don't care enough

**How to know you're failing:**
- Users don't get the point
- Have to explain it repeatedly
- "Why not just use X?" is common
- No organic growth

**How to prevent:**
- Validate problem with others FIRST
- Ship early, get feedback
- Be willing to pivot
- Kill project if no traction

---

### Failure Mode #4: **Maintenance Burnout** (50% probability)

**Why:**
- You have other priorities
- Issue backlog grows
- Breaking changes needed
- Community expects support

**How to know you're failing:**
- Issues > 50 open
- Last commit > 3 months ago
- PRs sitting unreviewed
- You dread opening GitHub

**How to prevent:**
- Set maintenance expectations
- Limit scope ruthlessly
- Accept breaking changes
- Consider co-maintainers early

---

### Failure Mode #5: **Platform Fragmentation** (30% probability)

**Why:**
- macOS updates break things
- Linux distros evolve
- Windows compatibility hard
- Testing matrix explodes

**How to know you're failing:**
- Issues are platform-specific
- Can't reproduce bugs
- Tests fail on different OSes
- Spend time on platform quirks

**How to prevent:**
- Minimal platform abstraction
- Let users handle platform quirks
- Good error messages
- Don't promise universal compatibility

---

## What You're Doing RIGHT

Let's be fair - you're avoiding many mistakes:

### ✅ Zero Dependencies
- No Python/Ruby/Node required
- Single binary
- Fast startup

### ✅ Simple Core
- ~1300 LOC currently
- Clear abstractions
- Easy to understand

### ✅ Declarative Config
- JSON (not DSL)
- Schema validated
- Version controlled

### ✅ Solving Real Pain
- You've hit this 100 times
- Real problem, not theoretical
- You'll use it yourself

### ✅ Platform Abstraction
- Cross-platform from day 1
- Simple detection
- Bare commands (no magic)

---

## What You're Doing WRONG (Probably)

Let's be brutally honest:

### ⚠️ Plugin System Risk

**The problem:**
- Plugins add complexity
- Plugin maintenance burden
- Plugin discovery problem
- Reinventing package managers

**Alternatives:**
- Just ship examples instead
- External tools via webhooks
- Composition via pipes

### ⚠️ Facts System Risk

**The problem:**
- Templating gets complex
- Facts need validation
- Debugging is hard
- Users don't think this way

**Alternatives:**
- Just use environment variables
- Let users run commands
- Keep it explicit

### ⚠️ No Killer Use Case

**The problem:**
- "Better dev setup" isn't compelling
- Competes with existing tools
- Not 10x better at anything

**What you need:**
- **One killer use case** where Sink is obviously the best
- Example: "Backstage service templates" or "systemd unit generation"
- Something no other tool does well

### ⚠️ Target Audience Unclear

**The problem:**
- Developers? (They write bash)
- Ops? (They use Ansible)
- DevOps? (They use Terraform)
- Students? (They use README)

**What you need:**
- Pick ONE audience first
- Get 100 of them using it
- Then expand

---

## The Hard Questions

### Question 1: **Is this just a better bash script?**

**If YES:**
- Why not make a bash framework instead?
- Sink.sh that sources helpers?
- No compilation, no binary

**If NO:**
- What does Sink do that bash can't?
- Is that difference worth learning a new tool?

### Question 2: **Is this just simpler Ansible?**

**If YES:**
- Why not use Ansible and simplify?
- Or contribute to Ansible?
- Or make Ansible templates?

**If NO:**
- What can Sink do that Ansible can't?
- Is that difference compelling?

### Question 3: **Who will maintain configs?**

**The problem:**
- Software changes (commands, flags, paths)
- Configs break
- Who updates them?

**Solutions:**
- Versioned configs?
- Config marketplace with ratings?
- Automated testing?

### Question 4: **Why will anyone share configs?**

**The problem:**
- NIH syndrome ("my setup is special")
- Company-specific configs
- No incentive to share

**Solutions:**
- Make sharing trivial (`sink publish`?)
- Build a community/forum?
- Gamification (badges, stars)?

---

## The Success Criteria

Let's be specific about what success looks like:

### 6 Months:
- ✅ 5 real users (not you)
- ✅ 10+ configs shared
- ✅ 100 GitHub stars
- ✅ 1 blog post by someone else
- ❌ Core < 2000 LOC
- ❌ Zero CVEs
- ❌ No dependencies added

### 1 Year:
- ✅ 50 real users
- ✅ 100+ configs shared
- ✅ 1000 GitHub stars
- ✅ Mentioned in "awesome" lists
- ✅ 5+ contributors
- ❌ Core still < 2000 LOC
- ❌ Used in 5+ production environments

### 2 Years:
- ✅ 500+ users
- ✅ 1000+ configs
- ✅ 5000 GitHub stars
- ✅ Conference talks
- ✅ Integration with major tools (Backstage, etc.)
- ❌ Core still maintainable by 1 person
- ❌ No major rewrites

**If these aren't hit: KILL THE PROJECT**

Don't let it become abandonware.

---

## The Uncomfortable Truth

### You're Probably Solving a Problem That Doesn't Exist

**Evidence:**
- Hundreds of projects tried
- All failed or stagnated
- People still use bash scripts
- Or they use Ansible

**Possible conclusions:**
1. **The problem isn't painful enough** - People tolerate bash scripts
2. **The solution space is wrong** - Need different approach
3. **Network effects too strong** - Ansible/Make have won
4. **The gap is illusory** - Maybe bash → Ansible is fine

**Counter-evidence:**
- You've needed this 100 times
- Every repo has setup.sh
- Onboarding is consistently painful
- CI/CD needs consistent setup

### The Only Way to Know: SHIP IT

**The test:**
1. Build minimal version (DONE ✅)
2. Share with 10 devs
3. Watch them try it
4. See if they use it again

**If they don't:**
- They'll tell you why
- That's your answer
- Pivot or kill

**If they do:**
- Find out what they love
- Double down on that
- Ignore everything else

---

## The Recommendation

### Phase 1: Validate (Now - 1 month)

1. **Pick ONE use case:**
   - "Setup dev environment for Node.js projects"
   - "Generate systemd units from configs"
   - "Backstage service templates"
   
2. **Build 10 configs for that use case**

3. **Find 10 people with that problem**

4. **Watch them use it**

5. **Measure:**
   - Did they succeed?
   - Did they use it again?
   - Did they share configs?

### Phase 2: Decide (1 month)

**If validation succeeds:**
- Double down
- Build community
- Iterate on feedback

**If validation fails:**
- Pivot to different use case
- Or simplify dramatically
- Or kill project

### Phase 3: Scale or Kill (6 months)

**If hitting success criteria:**
- Add features carefully
- Build ecosystem
- Maintain ruthlessly

**If not hitting criteria:**
- **KILL THE PROJECT**
- Don't let it rot
- Write a postmortem
- Move on

---

## The Truth You Don't Want to Hear

**Most likely outcome:**

You'll work on this for 3-6 months, get it to 80% complete, then:
- Get busy with other things
- Realize adoption is slow
- Find edge cases are hard
- Discover people don't care

And it'll sit on GitHub with 50 stars, last commit 2 years ago.

**Like 90% of side projects.**

**How to avoid this:**

1. **Set a deadline**: 6 months. Ship or kill.
2. **Set metrics**: 100 stars or kill.
3. **Actually kill it**: Don't let it rot.
4. **Write the postmortem**: Share what you learned.

---

## Conclusion: How to Not Fail

### The Rules:

1. **Solve ONE problem perfectly** (not 10 problems poorly)
2. **Stay small** (< 2000 LOC core forever)
3. **Say NO** (to 90% of feature requests)
4. **Ship fast** (validate in weeks, not months)
5. **Measure ruthlessly** (stars, users, usage)
6. **Kill quickly** (if not working, stop)
7. **No dependencies** (ever, for any reason)
8. **Clear target** (pick ONE audience)
9. **10x better** (at ONE specific thing)
10. **Use it yourself** (dogfood relentlessly)

### The One Thing That Could Make This Work:

**Find the ONE thing that Sink does that nothing else can do.**

Not "better than", but "ONLY Sink can do this."

Maybe that's:
- Backstage integration
- systemd generation
- Cross-platform + idempotent + zero-config
- Something you haven't thought of yet

**Find that thing, or this will fail.**

---

*"The graveyard is full of indispensable projects." - Unknown*
