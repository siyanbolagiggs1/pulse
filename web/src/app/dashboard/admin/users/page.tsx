"use client";
import { useEffect, useState } from "react";
import { adminApi } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Pagination } from "@/components/ui/pagination";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Trash2, Loader2, AlertTriangle } from "lucide-react";
import { format } from "date-fns";
import { toast } from "@/components/ui/use-toast";

interface AdminUser {
  id: string;
  email: string;
  name: string;
  role: string;
  trustScore: number;
  isSuspended: boolean;
  createdAt: string;
}

const PAGE_SIZE = 20;

export default function AdminUsersPage() {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [role, setRole] = useState("all");
  const [page, setPage] = useState(1);
  const [pages, setPages] = useState(1);
  const [acting, setActing] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<AdminUser | null>(null);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => { setPage(1); }, [role]);

  useEffect(() => {
    const params: Record<string, string | number> = { page, limit: PAGE_SIZE };
    if (role !== "all") params.role = role;
    setLoading(true);
    adminApi.listUsers(params)
      .then((r) => {
        setUsers(r.data.data);
        setPages(r.data.meta?.pages ?? 1);
      })
      .catch(() => toast({ title: "Failed to load users", variant: "destructive" }))
      .finally(() => setLoading(false));
  }, [role, page]);

  const handleSuspend = async (id: string) => {
    setActing(id);
    try {
      await adminApi.suspendUser(id);
      toast({ title: "User suspended" });
      setUsers((prev) => prev.map((u) => u.id === id ? { ...u, isSuspended: true } : u));
    } catch {
      toast({ title: "Action failed", variant: "destructive" });
    } finally { setActing(null); }
  };

  const handleUnsuspend = async (id: string) => {
    setActing(id);
    try {
      await adminApi.unsuspendUser(id);
      toast({ title: "User unsuspended" });
      setUsers((prev) => prev.map((u) => u.id === id ? { ...u, isSuspended: false, trustScore: 50 } : u));
    } catch {
      toast({ title: "Action failed", variant: "destructive" });
    } finally { setActing(null); }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await adminApi.deleteUser(deleteTarget.id);
      toast({ title: "User deleted" });
      setUsers((prev) => prev.filter((u) => u.id !== deleteTarget.id));
      setDeleteTarget(null);
    } catch (err: any) {
      toast({
        title: "Could not delete user",
        description: err?.response?.data?.message,
        variant: "destructive",
      });
    } finally { setDeleting(false); }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">Users</h2>
          <p className="text-muted-foreground">Manage platform users</p>
        </div>
        <Select value={role} onValueChange={setRole}>
          <SelectTrigger className="w-36"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Roles</SelectItem>
            <SelectItem value="user">User</SelectItem>
            <SelectItem value="admin">Admin</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-6 space-y-3">{[...Array(5)].map((_, i) => <Skeleton key={i} className="h-10" />)}</div>
          ) : users.length === 0 ? (
            <p className="py-12 text-center text-muted-foreground">No users found</p>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Email</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead>Trust Score</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Joined</TableHead>
                    <TableHead>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {users.map((u) => (
                    <TableRow key={u.id}>
                      <TableCell className="font-medium">{u.name}</TableCell>
                      <TableCell className="text-muted-foreground text-sm">{u.email}</TableCell>
                      <TableCell><Badge variant="secondary" className="capitalize">{u.role}</Badge></TableCell>
                      <TableCell>
                        <span className={u.trustScore < 30 ? "text-destructive" : u.trustScore < 60 ? "text-yellow-400" : "text-green-400"}>
                          {u.trustScore}
                        </span>
                      </TableCell>
                      <TableCell>
                        <Badge variant={u.isSuspended ? "destructive" : "success"}>
                          {u.isSuspended ? "Suspended" : "Active"}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-muted-foreground text-sm">{format(new Date(u.createdAt), "MMM d, yyyy")}</TableCell>
                      <TableCell>
                        {u.role !== "admin" && (
                          <div className="flex items-center gap-2">
                            {u.isSuspended ? (
                              <Button size="sm" variant="outline" onClick={() => handleUnsuspend(u.id)} disabled={acting === u.id}>
                                Unsuspend
                              </Button>
                            ) : (
                              <Button size="sm" variant="destructive" onClick={() => handleSuspend(u.id)} disabled={acting === u.id}>
                                Suspend
                              </Button>
                            )}
                            <Button
                              size="icon" variant="ghost" className="h-8 w-8 text-muted-foreground hover:text-destructive"
                              onClick={() => setDeleteTarget(u)}
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </div>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              <div className="px-6 pb-4">
                <Pagination page={page} pages={pages} onChange={setPage} />
              </div>
            </>
          )}
        </CardContent>
      </Card>

      <Dialog open={!!deleteTarget} onOpenChange={(o) => { if (!o) setDeleteTarget(null); }}>
        <DialogContent className="max-w-md">
          <DialogHeader><DialogTitle>Delete {deleteTarget?.name}?</DialogTitle></DialogHeader>
          <div className="flex gap-3 rounded-md bg-destructive/10 border border-destructive/30 p-3 text-sm">
            <AlertTriangle className="h-5 w-5 shrink-0 text-destructive" />
            <p>
              This permanently deletes this user and everything tied to them — wallet, transactions,
              campaigns/submissions, messages, connected accounts. This cannot be undone, and fails if
              their wallet balance isn&apos;t zero.
            </p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)} disabled={deleting}>Cancel</Button>
            <Button variant="destructive" onClick={handleDelete} disabled={deleting}>
              {deleting ? <><Loader2 className="mr-2 h-4 w-4 animate-spin" />Deleting…</> : "Yes, delete user"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
