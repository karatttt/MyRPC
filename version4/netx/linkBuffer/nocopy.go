
 package linkBuffer


// type Reader interface {
// 	// Next returns a slice containing the next n bytes from the buffer,
// 	// advancing the buffer as if the bytes had been returned by Read.
// 	//
// 	// If there are fewer than n bytes in the buffer, Next returns will be blocked
// 	// until data enough or an error occurs (such as a wait timeout).
// 	//
// 	// The slice p is only valid until the next call to the Release method.
// 	// Next is not globally optimal, and Skip, ReadString, ReadBinary methods
// 	// are recommended for specific scenarios.
// 	//
// 	// Return: len(p) must be n or 0, and p and error cannot be nil at the same time.
// 	Next(n int) (p []byte, err error)

// 	// Peek returns the next n bytes without advancing the reader.
// 	// Other behavior is the same as Next.
// 	Peek(n int) (buf []byte, err error)

// 	// Skip the next n bytes and advance the reader, which is
// 	// a faster implementation of Next when the next data is not used.
// 	Skip(n int) (err error)

// 	// Until reads until the first occurrence of delim in the input,
// 	// returning a slice stops with delim in the input buffer.
// 	// If Until encounters an error before finding a delimiter,
// 	// it returns all the data in the buffer and the error itself (often ErrEOF or ErrConnClosed).
// 	// Until returns err != nil only if line does not end in delim.
// 	Until(delim byte) (line []byte, err error)

// 	// ReadString is a faster implementation of Next when a string needs to be returned.
// 	// It replaces:
// 	//
// 	//  var p, err = Next(n)
// 	//  return string(p), err
// 	//
// 	ReadString(n int) (s string, err error)

// 	// ReadBinary is a faster implementation of Next when it needs to
// 	// return a copy of the slice that is not shared with the underlying layer.
// 	// It replaces:
// 	//
// 	//  var p, err = Next(n)
// 	//  var b = make([]byte, n)
// 	//  copy(b, p)
// 	//  return b, err
// 	//
// 	ReadBinary(n int) (p []byte, err error)

// 	// ReadByte is a faster implementation of Next when a byte needs to be returned.
// 	// It replaces:
// 	//
// 	//  var p, err = Next(1)
// 	//  return p[0], err
// 	//
// 	ReadByte() (b byte, err error)

// 	// Slice returns a new Reader containing the Next n bytes from this Reader.
// 	//
// 	// If you want to make a new Reader using the []byte returned by Next, Slice already does that,
// 	// and the operation is zero-copy. Besides, Slice would also Release this Reader.
// 	// The logic pseudocode is similar:
// 	//
// 	//  var p, err = this.Next(n)
// 	//  var reader = new Reader(p) // pseudocode
// 	//  this.Release()
// 	//  return reader, err
// 	//
// 	Slice(n int) (r Reader, err error)

// 	// Release the memory space occupied by all read slices. This method needs to be executed actively to
// 	// recycle the memory after confirming that the previously read data is no longer in use.
// 	// After invoking Release, the slices obtained by the method such as Next, Peek, Skip will
// 	// become an invalid address and cannot be used anymore.
// 	Release() (err error)

// 	// Len returns the total length of the readable data in the reader.
// 	Len() (length int)
// }

// // Writer is a collection of operations for nocopy writes.
// //
// // The usage of the design is a two-step operation, first apply for a section of memory,
// // fill it and then submit. E.g:
// //
// //	var buf, _ = Malloc(n)
// //	buf = append(buf[:0], ...)
// //	Flush()
// //
// // Note that it is not recommended to submit self-managed buffers to Writer.
// // Since the writer is processed asynchronously, if the self-managed buffer is used and recycled after submission,
// // it may cause inconsistent life cycle problems. Of course this is not within the scope of the design.
// type Writer interface {
// 	// Malloc returns a slice containing the next n bytes from the buffer,
// 	// which will be written after submission(e.g. Flush).
// 	//
// 	// The slice p is only valid until the next submit(e.g. Flush).
// 	// Therefore, please make sure that all data has been written into the slice before submission.
// 	Malloc(n int) (buf []byte, err error)

// 	// WriteString is a faster implementation of Malloc when a string needs to be written.
// 	// It replaces:
// 	//
// 	//  var buf, err = Malloc(len(s))
// 	//  n = copy(buf, s)
// 	//  return n, err
// 	//
// 	// The argument string s will be referenced based on the original address and will not be copied,
// 	// so make sure that the string s will not be changed.
// 	WriteString(s string) (n int, err error)

// 	// WriteBinary is a faster implementation of Malloc when a slice needs to be written.
// 	// It replaces:
// 	//
// 	//  var buf, err = Malloc(len(b))
// 	//  n = copy(buf, b)
// 	//  return n, err
// 	//
// 	// The argument slice b will be referenced based on the original address and will not be copied,
// 	// so make sure that the slice b will not be changed.
// 	WriteBinary(b []byte) (n int, err error)

// 	// WriteByte is a faster implementation of Malloc when a byte needs to be written.
// 	// It replaces:
// 	//
// 	//  var buf, _ = Malloc(1)
// 	//  buf[0] = b
// 	//
// 	WriteByte(b byte) (err error)

// 	// WriteDirect is used to insert an additional slice of data on the current write stream.
// 	// For example, if you plan to execute:
// 	//
// 	//  var bufA, _ = Malloc(nA)
// 	//  WriteBinary(b)
// 	//  var bufB, _ = Malloc(nB)
// 	//
// 	// It can be replaced by:
// 	//
// 	//  var buf, _ = Malloc(nA+nB)
// 	//  WriteDirect(b, nB)
// 	//
// 	// where buf[:nA] = bufA, buf[nA:nA+nB] = bufB.
// 	WriteDirect(p []byte, remainCap int) error

// 	// MallocAck will keep the first n malloc bytes and discard the rest.
// 	// The following behavior:
// 	//
// 	//  var buf, _ = Malloc(8)
// 	//  buf = buf[:5]
// 	//  MallocAck(5)
// 	//
// 	// equivalent as
// 	//  var buf, _ = Malloc(5)
// 	//
// 	MallocAck(n int) (err error)

// 	// Append the argument writer to the tail of this writer and set the argument writer to nil,
// 	// the operation is zero-copy, similar to p = append(p, w.p).
// 	Append(w Writer) (err error)

// 	// Flush will submit all malloc data and must confirm that the allocated bytes have been correctly assigned.
// 	// Its behavior is equivalent to the io.Writer hat already has parameters(slice b).
// 	Flush() (err error)

// 	// MallocLen returns the total length of the writable data that has not yet been submitted in the writer.
// 	MallocLen() (length int)
// }
