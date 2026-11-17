// Package scene builds and runs scenes for the Castrum engine. A scene is a
// [Scene] builder that turns props into a [Node] tree. Entering an instance
// spawns an entity for each node; exiting it destroys those entities.
//
// A [Director] queues requests to switch, load, update, or remove scene
// instances. Queuing does not apply a request. [NewSystem] applies the queue
// once per frame in FIFO order, so requests queued from a hook run in the
// next frame. Pending switches are coalesced, so only the latest one runs.
//
// [Director.Switch] replaces every loaded instance with a new one.
// [Director.Load] adds an instance alongside the current ones. Switch and
// update build the new content before touching the old, so a failed request
// leaves the live content in place. A failed request does not stop the other
// requests in its batch.
//
// Nodes can declare assets with [Warm] to load before entry. Enter hooks run
// after the tree spawns, and exit hooks run before its entities are destroyed.
package scene
