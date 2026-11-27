// Package graphtest is a recorded fake of the Microsoft Graph endpoints the msgraph adapters
// call (ADR-0023): the token endpoint, drive deltas, items, content and permissions, users'
// groups, managers and calendars, Teams meetings and transcripts, and shared mailboxes. Its
// shapes are Graph v1.0's; a run against a real tenant stays In review.
package graphtest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Token is the access token the fake hands out and expects.
const Token = "graph-token"

// Creds are credentials the fake accepts.
const Creds = `{"tenant_id":"t-acme","client_id":"app-1","client_secret":"s3cret"}`

// Item is one drive item.
type Item struct {
	ID, Name, Path, Owner, Label string // Path is the folder under the drive's root, "/Finance"
	Content                      []byte
	Version                      int
	Deleted, Folder              bool
	// Grants are user emails and group ids; Links are sharing link scopes (anonymous,
	// organization, users).
	Users, Groups, Links []string
}

// Meeting is one Teams meeting with its transcript.
type Meeting struct {
	ID, Organizer, Subject, Start string
	Attendees                     []string // UPNs
	Transcript                    string   // VTT; "" for none
}

// Event is one calendar event.
type Event struct {
	ID, Subject, Body, Start string
	Attendees                []string
}

// Message is one mail message in a mailbox.
type Message struct {
	ID, From, Subject, Body string
	Attachments             map[string][]byte
}

// Fake is the recorded Graph.
type Fake struct {
	Server *httptest.Server

	mu       sync.Mutex
	Drives   map[string]map[string]*Item // drive -> item id -> item
	log      map[string][]string         // drive -> item ids in change order
	shared   map[string]map[int]bool     // drive -> log positions that only changed sharing
	failNext string                      // the next permissions calls: 500, timeout or 404
	failLeft int                         // how many of them
	Groups   map[string][]string         // group id -> member emails or group ids ("g:ID")
	Mails    map[string]string           // group id -> its mail
	Managers map[string]string           // email -> manager email
	Users    map[string]string           // user id -> email
	Meetings map[string][]*Meeting       // organizer user id -> meetings
	Events   map[string][]Event          // email -> events
	Boxes    map[string][]Message        // mailbox -> messages
	// Calls are the paths asked for, in order.
	Calls []string
}

// New starts a fake with no data.
func New() *Fake {
	f := &Fake{Drives: map[string]map[string]*Item{}, log: map[string][]string{}, shared: map[string]map[int]bool{}, Groups: map[string][]string{}, Mails: map[string]string{},
		Managers: map[string]string{}, Users: map[string]string{}, Meetings: map[string][]*Meeting{}, Events: map[string][]Event{}, Boxes: map[string][]Message{}}
	f.Server = httptest.NewServer(f)
	return f
}

// Base is the fake's Graph base; Login its sign-in.
func (f *Fake) Base() string  { return f.Server.URL + "/v1.0" }
func (f *Fake) Login() string { return f.Server.URL + "/login" }

// Put adds or changes an item, and logs the change.
func (f *Fake) Put(drive string, it Item) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Drives[drive] == nil {
		f.Drives[drive] = map[string]*Item{}
	}
	if old := f.Drives[drive][it.ID]; old != nil && it.Version == 0 {
		it.Version = old.Version + 1
	}
	if it.Version == 0 {
		it.Version = 1
	}
	f.Drives[drive][it.ID] = &it
	f.log[drive] = append(f.log[drive], it.ID)
}

// Share changes only who an item is shared with, as a sharing change in the delta.
func (f *Fake) Share(drive, id string, users ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if it := f.Drives[drive][id]; it != nil {
		it.Users = users
		if f.shared[drive] == nil {
			f.shared[drive] = map[int]bool{}
		}
		f.shared[drive][len(f.log[drive])] = true
		f.log[drive] = append(f.log[drive], id)
	}
}

// FailNext makes the next permissions call fail: "500", "timeout" or "404".
func (f *Fake) FailNext(kind string) { f.FailCalls(kind, 1) }

// FailCalls makes the next n permissions calls fail as FailNext does.
func (f *Fake) FailCalls(kind string, n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failNext, f.failLeft = kind, n
}

// failing takes one of the failures FailCalls set, or "".
func (f *Fake) failing() string {
	if f.failLeft <= 0 {
		return ""
	}
	f.failLeft--
	return f.failNext
}

// Delete marks an item deleted, and logs the change.
func (f *Fake) Delete(drive, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if it := f.Drives[drive][id]; it != nil {
		it.Deleted = true
		f.log[drive] = append(f.log[drive], id)
	}
}

func (f *Fake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, r.URL.Path)
	if strings.HasPrefix(r.URL.Path, "/login/") {
		if err := r.ParseForm(); err != nil || r.Form.Get("client_secret") != "s3cret" || r.Form.Get("grant_type") != "client_credentials" {
			http.Error(w, `{"error":"invalid_client"}`, http.StatusUnauthorized)
			return
		}
		writeJSON(w, map[string]any{"access_token": Token, "expires_in": 3600})
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+Token {
		http.Error(w, `{"error":{"code":"InvalidAuthenticationToken"}}`, http.StatusUnauthorized)
		return
	}
	p := strings.TrimPrefix(r.URL.Path, "/v1.0")
	for _, rt := range routes {
		if m := rt.re.FindStringSubmatch(p); m != nil {
			rt.h(f, w, r, m[1:])
			return
		}
	}
	http.Error(w, `{"error":{"code":"itemNotFound"}}`, http.StatusNotFound)
}

type route struct {
	re *regexp.Regexp
	h  func(f *Fake, w http.ResponseWriter, r *http.Request, m []string)
}

var routes = []route{
	{regexp.MustCompile(`^/drives/([^/]+)/root/delta$`), (*Fake).delta},
	{regexp.MustCompile(`^/drives/([^/]+)/items/([^/]+)$`), (*Fake).item},
	{regexp.MustCompile(`^/drives/([^/]+)/items/([^/]+)/content$`), (*Fake).content},
	{regexp.MustCompile(`^/drives/([^/]+)/items/([^/]+)/permissions$`), (*Fake).permissions},
	{regexp.MustCompile(`^/users/([^/]+)/transitiveMemberOf$`), (*Fake).memberOf},
	{regexp.MustCompile(`^/groups/([^/]+)/transitiveMembers$`), (*Fake).members},
	{regexp.MustCompile(`^/users/([^/]+)/manager$`), (*Fake).manager},
	{regexp.MustCompile(`^/users/([^/]+)/calendarView$`), (*Fake).calendar},
	{regexp.MustCompile(`^/users/([^/]+)/onlineMeetings/getAllTranscripts\(meetingOrganizerUserId='[^']+'\)/delta$`), (*Fake).transcripts},
	{regexp.MustCompile(`^/users/([^/]+)/onlineMeetings/([^/]+)$`), (*Fake).meeting},
	{regexp.MustCompile(`^/users/([^/]+)/onlineMeetings/([^/]+)/transcripts/([^/]+)/content$`), (*Fake).transcript},
	{regexp.MustCompile(`^/users/([^/]+)/mailFolders/inbox/messages/delta$`), (*Fake).mail},
	{regexp.MustCompile(`^/users/([^/]+)/messages/([^/]+)/attachments$`), (*Fake).attachments},
	{regexp.MustCompile(`^/users/([^/]+)$`), (*Fake).user},
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func notFound(w http.ResponseWriter) {
	http.Error(w, `{"error":{"code":"itemNotFound"}}`, http.StatusNotFound)
}

func (f *Fake) link(path string, q url.Values) string {
	return f.Base() + path + "?" + q.Encode()
}

func (f *Fake) itemJSON(drive string, it *Item) map[string]any {
	out := map[string]any{"id": it.ID, "name": it.Name}
	if it.Deleted {
		out["deleted"] = map[string]string{"state": "deleted"}
		return out
	}
	if it.Folder {
		out["folder"] = map[string]int{"childCount": 0}
	} else {
		out["file"] = map[string]string{"mimeType": "application/octet-stream"}
	}
	out["cTag"] = fmt.Sprintf(`"c:{%s},%d"`, it.ID, it.Version)
	out["size"] = len(it.Content)
	out["parentReference"] = map[string]string{"driveId": drive, "path": "/drives/" + drive + "/root:" + it.Path}
	if it.Owner != "" {
		out["createdBy"] = map[string]any{"user": map[string]string{"email": it.Owner}}
	}
	if it.Label != "" {
		out["sensitivityLabel"] = map[string]string{"labelId": it.Label}
	}
	return out
}

// delta lists the changes after token (or every item with no token), then a deltaLink.
func (f *Fake) delta(w http.ResponseWriter, r *http.Request, m []string) {
	drive := m[0]
	from := 0
	if t := r.URL.Query().Get("token"); t != "" {
		from, _ = strconv.Atoi(t)
	}
	var vals []map[string]any
	seen := map[string]bool{}
	log := f.log[drive]
	sharing := r.Header.Get("Prefer") == "deltashowsharingchanges"
	for i := len(log) - 1; i >= from; i-- {
		id := log[i]
		if seen[id] {
			continue
		}
		seen[id] = true
		v := f.itemJSON(drive, f.Drives[drive][id])
		if sharing && f.shared[drive][i] {
			v["@microsoft.graph.sharedChanged"] = "True"
		}
		vals = append([]map[string]any{v}, vals...)
	}
	writeJSON(w, map[string]any{"value": vals, "@odata.deltaLink": f.link("/drives/"+drive+"/root/delta", url.Values{"token": {strconv.Itoa(len(log))}})})
}

func (f *Fake) item(w http.ResponseWriter, _ *http.Request, m []string) {
	it := f.Drives[m[0]][m[1]]
	if it == nil || it.Deleted {
		notFound(w)
		return
	}
	writeJSON(w, f.itemJSON(m[0], it))
}

func (f *Fake) content(w http.ResponseWriter, _ *http.Request, m []string) {
	it := f.Drives[m[0]][m[1]]
	if it == nil || it.Deleted {
		notFound(w)
		return
	}
	_, _ = w.Write(it.Content)
}

func (f *Fake) permissions(w http.ResponseWriter, _ *http.Request, m []string) {
	switch f.failing() {
	case "500":
		http.Error(w, `{"error":{"code":"generalException"}}`, http.StatusInternalServerError)
		return
	case "404":
		notFound(w)
		return
	case "timeout":
		f.mu.Unlock()
		time.Sleep(900 * time.Millisecond)
		f.mu.Lock()
	}
	it := f.Drives[m[0]][m[1]]
	if it == nil || it.Deleted {
		notFound(w)
		return
	}
	var vals []map[string]any
	for _, u := range it.Users {
		vals = append(vals, map[string]any{"roles": []string{"read"}, "grantedToV2": map[string]any{"user": map[string]string{"email": u}}})
	}
	for _, g := range it.Groups {
		vals = append(vals, map[string]any{"roles": []string{"read"}, "grantedToV2": map[string]any{"group": map[string]string{"id": g, "email": f.Mails[g]}}})
	}
	for _, l := range it.Links {
		vals = append(vals, map[string]any{"roles": []string{"read"}, "link": map[string]string{"scope": l, "type": "view"}})
	}
	writeJSON(w, map[string]any{"value": vals})
}

// memberOf is a user's groups, nested ones included.
func (f *Fake) memberOf(w http.ResponseWriter, _ *http.Request, m []string) {
	who := strings.ToLower(m[0])
	var vals []map[string]any
	seen := map[string]bool{}
	var walk func(string)
	walk = func(member string) {
		for g, ms := range f.Groups {
			if seen[g] {
				continue
			}
			for _, x := range ms {
				if strings.ToLower(x) == member {
					seen[g] = true
					vals = append(vals, map[string]any{"@odata.type": "#microsoft.graph.group", "id": g, "mail": f.Mails[g]})
					walk("g:" + g)
				}
			}
		}
	}
	walk(who)
	writeJSON(w, map[string]any{"value": vals})
}

// members is a group's members, nested groups' members included, in pages of two.
func (f *Fake) members(w http.ResponseWriter, r *http.Request, m []string) {
	var users []string
	seen := map[string]bool{}
	var walk func(string)
	walk = func(g string) {
		for _, x := range f.Groups[g] {
			if strings.HasPrefix(x, "g:") {
				walk(strings.TrimPrefix(x, "g:"))
			} else if !seen[x] {
				seen[x] = true
				users = append(users, x)
			}
		}
	}
	walk(m[0])
	skip, _ := strconv.Atoi(r.URL.Query().Get("skip"))
	end := skip + 2
	if end > len(users) {
		end = len(users)
	}
	var vals []map[string]any
	for _, u := range users[skip:end] {
		vals = append(vals, map[string]any{"@odata.type": "#microsoft.graph.user", "mail": u})
	}
	out := map[string]any{"value": vals}
	if end < len(users) {
		out["@odata.nextLink"] = f.link("/groups/"+m[0]+"/transitiveMembers", url.Values{"skip": {strconv.Itoa(end)}})
	}
	writeJSON(w, out)
}

func (f *Fake) manager(w http.ResponseWriter, _ *http.Request, m []string) {
	mgr, ok := f.Managers[strings.ToLower(m[0])]
	if !ok {
		notFound(w)
		return
	}
	writeJSON(w, map[string]string{"mail": mgr})
}

func (f *Fake) user(w http.ResponseWriter, _ *http.Request, m []string) {
	e, ok := f.Users[m[0]]
	if !ok {
		notFound(w)
		return
	}
	writeJSON(w, map[string]string{"id": m[0], "mail": e, "userPrincipalName": e})
}

func (f *Fake) calendar(w http.ResponseWriter, r *http.Request, m []string) {
	if r.URL.Query().Get("startDateTime") == "" || r.URL.Query().Get("endDateTime") == "" {
		http.Error(w, `{"error":{"code":"ErrorInvalidParameter"}}`, http.StatusBadRequest)
		return
	}
	var vals []map[string]any
	for _, e := range f.Events[strings.ToLower(m[0])] {
		var at []map[string]any
		for _, a := range e.Attendees {
			at = append(at, map[string]any{"emailAddress": map[string]string{"address": a}, "status": map[string]string{"response": "accepted"}})
		}
		vals = append(vals, map[string]any{"id": e.ID, "subject": e.Subject, "bodyPreview": e.Body,
			"start": map[string]string{"dateTime": e.Start, "timeZone": "UTC"}, "attendees": at})
	}
	writeJSON(w, map[string]any{"value": vals})
}

func (f *Fake) transcripts(w http.ResponseWriter, _ *http.Request, m []string) {
	var vals []map[string]any
	for _, mt := range f.Meetings[m[0]] {
		if mt.Transcript == "" {
			continue // a meeting without a transcript has none to list
		}
		vals = append(vals, map[string]any{"id": "tr-" + mt.ID, "meetingId": mt.ID, "meetingOrganizerId": m[0], "createdDateTime": mt.Start})
	}
	writeJSON(w, map[string]any{"value": vals, "@odata.deltaLink": f.Base() + "/users/" + m[0] + "/onlineMeetings/getAllTranscripts(meetingOrganizerUserId='" + m[0] + "')/delta?token=1"})
}

func (f *Fake) findMeeting(org, id string) *Meeting {
	for _, mt := range f.Meetings[org] {
		if mt.ID == id {
			return mt
		}
	}
	return nil
}

func (f *Fake) meeting(w http.ResponseWriter, _ *http.Request, m []string) {
	mt := f.findMeeting(m[0], m[1])
	if mt == nil {
		notFound(w)
		return
	}
	var at []map[string]any
	for _, a := range mt.Attendees {
		at = append(at, map[string]any{"upn": a})
	}
	writeJSON(w, map[string]any{"id": mt.ID, "subject": mt.Subject, "startDateTime": mt.Start,
		"participants": map[string]any{"organizer": map[string]string{"upn": mt.Organizer}, "attendees": at}})
}

func (f *Fake) transcript(w http.ResponseWriter, r *http.Request, m []string) {
	mt := f.findMeeting(m[0], m[1])
	if mt == nil || mt.Transcript == "" || "tr-"+mt.ID != m[2] || r.URL.Query().Get("$format") != "text/vtt" {
		notFound(w)
		return
	}
	_, _ = w.Write([]byte(mt.Transcript))
}

func (f *Fake) mail(w http.ResponseWriter, r *http.Request, m []string) {
	box := strings.ToLower(m[0])
	msgs, ok := f.Boxes[box]
	if !ok {
		http.Error(w, `{"error":{"code":"ErrorAccessDenied"}}`, http.StatusForbidden)
		return
	}
	from, _ := strconv.Atoi(r.URL.Query().Get("token"))
	var vals []map[string]any
	for _, msg := range msgs[min(from, len(msgs)):] {
		vals = append(vals, map[string]any{"id": msg.ID, "subject": msg.Subject, "body": map[string]string{"contentType": "text", "content": msg.Body},
			"from": map[string]any{"emailAddress": map[string]string{"address": msg.From}}, "hasAttachments": len(msg.Attachments) > 0})
	}
	writeJSON(w, map[string]any{"value": vals, "@odata.deltaLink": f.link("/users/"+box+"/mailFolders/inbox/messages/delta", url.Values{"token": {strconv.Itoa(len(msgs))}})})
}

func (f *Fake) attachments(w http.ResponseWriter, _ *http.Request, m []string) {
	for _, msg := range f.Boxes[strings.ToLower(m[0])] {
		if msg.ID != m[1] {
			continue
		}
		var vals []map[string]any
		for name, b := range msg.Attachments {
			vals = append(vals, map[string]any{"@odata.type": "#microsoft.graph.fileAttachment", "name": name, "contentBytes": b})
		}
		writeJSON(w, map[string]any{"value": vals})
		return
	}
	notFound(w)
}
